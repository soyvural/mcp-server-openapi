// Package main implements a simple task manager HTTP API for MCP server demonstration.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Task represents a task in the task manager.
type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskStore holds tasks in memory with thread-safe access.
type TaskStore struct {
	mu        sync.Mutex
	tasks     map[int]*Task
	idCounter int
}

// NewTaskStore creates a new task store with sample data.
func NewTaskStore() *TaskStore {
	store := &TaskStore{
		tasks:     make(map[int]*Task),
		idCounter: 1,
	}

	store.tasks[1] = &Task{
		ID:          1,
		Title:       "Setup development environment",
		Description: "Install Go, configure editor, clone repository",
		Completed:   true,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
	}
	store.idCounter = 2

	return store
}

func (s *TaskStore) list() []*Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (s *TaskStore) get(id int) (*Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	return task, ok
}

func (s *TaskStore) create(task *Task) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.ID = s.idCounter
	task.CreatedAt = time.Now()
	s.tasks[task.ID] = task
	s.idCounter++

	return task
}

func (s *TaskStore) update(id int, updates *Task) (*Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return nil, false
	}

	task.Title = updates.Title
	task.Description = updates.Description
	task.Completed = updates.Completed

	return task, true
}

func (s *TaskStore) delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.tasks[id]
	if ok {
		delete(s.tasks, id)
	}
	return ok
}

func main() {
	store := NewTaskStore()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /tasks", func(w http.ResponseWriter, r *http.Request) {
		tasks := store.list()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tasks)
	})

	mux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		var task Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		created := store.create(&task)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)
	})

	mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid task ID", http.StatusBadRequest)
			return
		}

		task, ok := store.get(id)
		if !ok {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	})

	mux.HandleFunc("PUT /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid task ID", http.StatusBadRequest)
			return
		}

		var updates Task
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		task, ok := store.update(id, &updates)
		if !ok {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	})

	mux.HandleFunc("DELETE /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid task ID", http.StatusBadRequest)
			return
		}

		if !store.delete(id) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	addr := ":9090"
	log.Printf("Task Manager API starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
