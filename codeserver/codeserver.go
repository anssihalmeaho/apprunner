package codeserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// CodeStore represents storage API
type CodeStore interface {
	Open() error
	Put(name string, content []byte) error
	GetByName(name string) ([]byte, bool)
	GetAll() []string
	DelByName(name string)
	Close()
}

// CodeServer represents codeserver
type CodeServer struct {
	store CodeStore
}

func (cs *CodeServer) handlePost(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	filename := pathParts[len(pathParts)-1]
	err = cs.store.Put(filename, body)
	if err != nil {
		log.Printf("Error in storing package: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (cs *CodeServer) handleGetAll(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	if name != "" {
		_, found := cs.store.GetByName(name)
		if !found {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		l := []string{name}
		resp, err := json.Marshal(&l)
		if err != nil {
			log.Printf("Error in reading packages: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
		return
	}

	packs := cs.store.GetAll()
	resp, err := json.Marshal(&packs)
	if err != nil {
		log.Printf("Error in reading packages: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (cs *CodeServer) handleGet(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	name := pathParts[len(pathParts)-1]
	if name == "" {
		http.Error(w, "assuming package name", http.StatusBadRequest)
		return
	}

	content, found := cs.store.GetByName(name)
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write(content)
}

func (cs *CodeServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	name := pathParts[len(pathParts)-1]
	if name == "" {
		http.Error(w, "assuming package name", http.StatusBadRequest)
		return
	}
	cs.store.DelByName(name)
}

// NewCodeServer ...
func NewCodeServer(store CodeStore) *CodeServer {
	return &CodeServer{store: store}
}

// GetHandler gets handler
func (cs *CodeServer) GetHandler() (hCol, hRes func(w http.ResponseWriter, r *http.Request)) {
	hCol = func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			cs.handleGetAll(w, r)
		default:
			http.Error(w, fmt.Sprintf("Unsupported method: %s", r.Method), http.StatusMethodNotAllowed)
		}
	}
	hRes = func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			cs.handlePost(w, r)
		case "GET":
			cs.handleGet(w, r)
		case "DELETE":
			cs.handleDelete(w, r)
		default:
			http.Error(w, fmt.Sprintf("Unsupported method: %s", r.Method), http.StatusMethodNotAllowed)
		}
	}
	return
}
