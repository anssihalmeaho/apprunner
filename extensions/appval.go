package extensions

import (
	"sync"

	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/funl/std"
)

type appvalStore struct {
	m map[string]map[string]funl.Value
	sync.RWMutex
}

func newAppvalStore() *appvalStore {
	return &appvalStore{
		m: make(map[string]map[string]funl.Value),
	}
}

func (av *appvalStore) Put(name, token string, value funl.Value) {
	av.Lock()
	defer av.Unlock()

	innerm, found := av.m[token]
	if !found {
		innerm = make(map[string]funl.Value)
		av.m[token] = innerm
	}
	av.m[token][name] = value
}

func (av *appvalStore) Get(name, token string) (funl.Value, bool) {
	av.RLock()
	defer av.RUnlock()

	v, found := av.m[token][name]
	if !found {
		v = funl.Value{Kind: funl.StringValue, Data: ""}
	}
	return v, found
}

var appValStore *appvalStore

func init() {
	appValStore = newAppvalStore()
	funl.AddExtensionInitializer(initAppval)
}

func initAppval(interpreter *funl.Interpreter) (err error) {
	stdModuleName := "appval"
	topFrame := funl.NewTopFrameWithInterpreter(interpreter)
	stdFuncs := []std.StdFuncInfo{
		{
			Name:   "getval",
			Getter: getGetval,
		},
		{
			Name:   "setval",
			Getter: getSetval,
		},
	}
	err = std.SetSTDFunctions(topFrame, stdModuleName, stdFuncs, interpreter)
	return
}

func getGetval(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 2 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need two", name, l)
		}
		if arguments[0].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}
		if arguments[1].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}
		tokenv := arguments[0].Data.(string)
		namev := arguments[1].Data.(string)
		val, found := appValStore.Get(namev, tokenv)

		values := []funl.Value{
			{
				Kind: funl.BoolValue,
				Data: found,
			},
			val,
		}
		retVal = funl.MakeListOfValues(frame, values)
		return
	}
}

func getSetval(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 3 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need three", name, l)
		}
		if arguments[0].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}
		if arguments[1].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}
		tokenv := arguments[0].Data.(string)
		namev := arguments[1].Data.(string)
		appValStore.Put(namev, tokenv, arguments[2])
		retVal = funl.Value{Kind: funl.BoolValue, Data: true}
		return
	}
}
