package executor

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/funl/std"
)

const defaultExitingTimeout = 20 // seconds

type app struct {
	id     int
	name   string
	exitCh chan (funl.Value)
}

type appStore struct {
	m    map[int]*app
	lock sync.RWMutex
}

func (aps *appStore) getAll() []*app {
	apps := []*app{}

	aps.lock.RLock()
	defer aps.lock.RUnlock()

	for _, appData := range aps.m {
		apps = append(apps, appData)
	}
	return apps
}

func (aps *appStore) stop(appID int) error {
	aps.lock.Lock()
	appInstance, found := aps.m[appID]
	aps.lock.Unlock()

	if !found {
		return fmt.Errorf("app not found")
	}
	if appInstance.exitCh == nil {
		return nil
	}
	appInstance.exitCh <- funl.Value{Kind: funl.StringValue, Data: "exit-from-user"}
	select {
	case <-appInstance.exitCh:
	case <-time.After(defaultExitingTimeout * time.Second):
	}
	return nil
}

func (aps *appStore) add(appInstance *app) error {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	aps.m[appInstance.id] = appInstance
	return nil
}

func (aps *appStore) del(appInstance *app) error {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	delete(aps.m, appInstance.id)
	return nil
}

func newArgEvaluator() *argEval {
	interpreter := funl.NewInterpreter()
	if err := std.InitSTD(interpreter); err != nil {
		panic(fmt.Errorf("Error in std-lib init (%v)", err))
	}
	if err := funl.InitFunSourceSTD(interpreter); err != nil {
		panic(fmt.Errorf("Error in std-lib (fun source) init (%v)", err))
	}
	frame := funl.NewTopFrameWithInterpreter(interpreter)
	frame.SetInProcCall(true)
	evaluatorItem := &funl.Item{
		Type: funl.ValueItem,
		Data: funl.Value{
			Kind: funl.StringValue,
			Data: "call(proc() import stdjson stdjson.decode end)",
		},
	}
	evaluator := funl.HandleEvalOP(frame, []*funl.Item{evaluatorItem})

	return &argEval{
		argEvaluator: evaluator,
		frame:        frame,
	}
}

type argEval struct {
	argEvaluator funl.Value
	frame        *funl.Frame
}

func newAppStore() *appStore {
	return &appStore{
		m: map[int]*app{},
	}
}

type packRunner struct {
	csAddr     string
	packGetter func(string) ([]byte, bool)
	idCount    int
	appstore   *appStore
	argsEval   *argEval
}

func (runner *packRunner) handleGetAll(w http.ResponseWriter, r *http.Request) {
	appsResp := []map[string]interface{}{}
	for _, app := range runner.appstore.getAll() {
		appInfo := map[string]interface{}{
			"id":   app.id,
			"name": app.name,
		}
		appsResp = append(appsResp, appInfo)
	}
	resp, err := json.Marshal(&appsResp)
	if err != nil {
		log.Printf("Error in reading apps: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (runner *packRunner) handleDelete(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	appID := pathParts[len(pathParts)-1]
	if appID == "" {
		http.Error(w, "assuming app name", http.StatusBadRequest)
		return
	}
	appIDNum, err := strconv.Atoi(appID)
	if err != nil {
		http.Error(w, "invalid app id", http.StatusBadRequest)
		return
	}
	err = runner.appstore.stop(appIDNum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (runner *packRunner) handleAppCreate(w http.ResponseWriter, r *http.Request) {
	type runRequest struct {
		Name           string          `json:"name"`
		Pack           string          `json:"pack"`
		Args           json.RawMessage `json:"args"`
		HaveCTXasLast  bool            `json:"ctx-last"`
		HaveCTXasFirst bool            `json:"ctx-1st"`
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req runRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var code []byte
	var packFound bool
	if runner.csAddr == "" {
		code, packFound = runner.packGetter(req.Pack)
	} else {
		code, err = func() ([]byte, error) {
			client := &http.Client{}
			resp, err := client.Get(fmt.Sprintf("http://%s/packs/%s", runner.csAddr, req.Pack))
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode != 200 {
				return nil, fmt.Errorf("Error from codeserver: %d: %s", resp.StatusCode, string(body))
			}
			return body, nil
		}()
		if err != nil {
			log.Printf("Error in getting package: %v", err)
		} else {
			packFound = true
		}
	}
	if !packFound {
		http.Error(w, "package not found", http.StatusNotFound)
		return
	}

	// Decode arguments
	item := &funl.Item{
		Type: funl.ValueItem,
		Data: funl.Value{
			Kind: funl.OpaqueValue,
			Data: std.NewOpaqueByteArray(req.Args),
		},
	}
	operands := []*funl.Item{
		&funl.Item{
			Type: funl.ValueItem,
			Data: runner.argsEval.argEvaluator,
		},
		item,
	}
	argListVal := funl.HandleCallOP(runner.argsEval.frame, operands)
	if argListVal.Kind != funl.ListValue {
		http.Error(w, "arguments should be in array", http.StatusBadRequest)
		return
	}
	resit := funl.NewListIterator(argListVal)
	resv := resit.Next()
	if (*resv).Kind != funl.BoolValue || !(*resv).Data.(bool) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	resv = resit.Next()
	resv = resit.Next()
	lit := funl.NewListIterator(*resv)
	args := []*funl.Item{}
	for {
		nextArg := lit.Next()
		if nextArg == nil {
			break
		}
		args = append(args, &funl.Item{Type: funl.ValueItem, Data: *nextArg})
	}

	// create app instance
	runner.idCount++
	appInstance := &app{
		id:   runner.idCount,
		name: req.Name,
	}
	runner.appstore.add(appInstance)

	cargs := []*funl.Item{}
	if req.HaveCTXasLast || req.HaveCTXasFirst {
		// add also context map
		channel := make(chan funl.Value)
		chanVal := funl.Value{Kind: funl.ChanValue, Data: channel}
		appInstance.exitCh = channel

		loggerProc := func(frame *funl.Frame, ops []funl.Value) funl.Value {
			s := fmt.Sprintf("app %d (%s):", appInstance.id, appInstance.name)
			largs := []interface{}{s}
			for _, v := range ops {
				largs = append(largs, v)
			}
			fmt.Println(largs...)
			return funl.Value{Kind: funl.BoolValue, Data: true}
		}
		operands = []*funl.Item{
			&funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.StringValue,
					Data: "exit-chan",
				},
			},
			&funl.Item{
				Type: funl.ValueItem,
				Data: chanVal,
			},
			&funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.StringValue,
					Data: "id",
				},
			},
			&funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.StringValue,
					Data: fmt.Sprintf("%d", appInstance.id),
				},
			},
			&funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.StringValue,
					Data: "name",
				},
			},
			&funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.StringValue,
					Data: appInstance.name,
				},
			},
			&funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.StringValue,
					Data: "log",
				},
			},
			&funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.ExtProcValue,
					Data: funl.ExtProcType{Impl: loggerProc},
				},
			},
		}
		mapv := funl.HandleMapOP(runner.argsEval.frame, operands)
		if req.HaveCTXasFirst {
			// add ctx as first argument
			cargs = append([]*funl.Item{&funl.Item{Type: funl.ValueItem, Data: mapv}}, args...)
		} else {
			// add ctx as last argument
			cargs = append(args, &funl.Item{Type: funl.ValueItem, Data: mapv})
		}
	} else {
		// no ctx given as argument
		cargs = args
	}

	// run app in own goroutine and interpreter
	go func(thisApp *app) {
		defer func() {
			if thisApp.exitCh != nil {
				close(thisApp.exitCh)
			}
		}()
		defer func() {
			if r := recover(); r != nil {
				fmt.Println(fmt.Sprintf("App runtime error:  %d (%s): %v", thisApp.id, thisApp.name, r))
			}
		}()
		defer runner.appstore.del(thisApp)

		retval, err := funl.FunlMainWithPackageContent(code, cargs, "main", req.Pack, std.InitSTD)
		if err != nil {
			panic(err)
		}

		fmt.Println(fmt.Sprintf("App exit: %d (%s): %#v", thisApp.id, thisApp.name, retval))
	}(appInstance)

	response := map[string]interface{}{
		"id": fmt.Sprintf("%d", appInstance.id),
	}
	resp, err := json.Marshal(&response)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

// GetHandler gets handler
func GetHandler(csAddr string, packGetter func(string) ([]byte, bool)) (hCol, hRes func(w http.ResponseWriter, r *http.Request)) {
	funl.PrintingRTElocationAndScopeEnabled = true
	server := &packRunner{
		csAddr:     csAddr,
		packGetter: packGetter,
		idCount:    10,
		appstore:   newAppStore(),
		argsEval:   newArgEvaluator(),
	}

	hCol = func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			server.handleAppCreate(w, r)
		case "GET":
			server.handleGetAll(w, r)
		default:
			http.Error(w, fmt.Sprintf("Unsupported method: %s", r.Method), http.StatusMethodNotAllowed)
		}
	}
	hRes = func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "DELETE":
			server.handleDelete(w, r)
		default:
			http.Error(w, fmt.Sprintf("Unsupported method: %s", r.Method), http.StatusMethodNotAllowed)
		}
	}
	return
}
