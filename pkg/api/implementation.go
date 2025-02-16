package api

import (
	"encoding/json"
	"net/http"
	"sync"
)

var cacheMx sync.RWMutex

// ensure that we've conformed to the `ServerInterface` with a compile-time check
var _ ServerInterface = (*TodoApiServer)(nil)

type TodoApiServer struct {
	cache map[string]Todo
}

func NewTodoApiServer() TodoApiServer {
	return TodoApiServer{
		cache: make(map[string]Todo),
	}
}

func (s TodoApiServer) GetTodos(w http.ResponseWriter, _ *http.Request) {
	cacheMx.RLock()

	tasks := make([]Todo, len(s.cache))
	i := 0
	for _, todo := range s.cache {
		tasks[i] = todo
		i++
	}

	cacheMx.RUnlock()

	payload, err := json.Marshal(tasks)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(payload); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s TodoApiServer) AddTodo(w http.ResponseWriter, r *http.Request) {
	cacheMx.Lock()
	defer cacheMx.Unlock()

	var todo Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, ok := s.cache[todo.Name]; ok {
		w.WriteHeader(http.StatusConflict)
		return
	}

	s.cache[todo.Name] = todo
	w.WriteHeader(http.StatusCreated)
}

func (s TodoApiServer) DeleteTodo(w http.ResponseWriter, r *http.Request, name string) {
	cacheMx.Lock()
	defer cacheMx.Unlock()

	if _, ok := s.cache[name]; !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	delete(s.cache, name)
	w.WriteHeader(http.StatusNoContent)
}

func (s TodoApiServer) UpdateTodo(w http.ResponseWriter, r *http.Request, name string) {
	cacheMx.Lock()
	defer cacheMx.Unlock()

	if _, ok := s.cache[name]; !ok {
		w.WriteHeader(http.StatusNotFound)
	}

	var todo Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.cache[name] = todo
	w.WriteHeader(http.StatusOK)
}
