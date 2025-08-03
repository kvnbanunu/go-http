package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type User struct {
	Name string `json:"name"`
}

var userCache = make(map[int]User)

// mux will spawn a go routine for each req, but accessing data is not threadsafe
// so a mutex is needed
var cacheMutex sync.RWMutex

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)

	mux.HandleFunc("POST /users", createUser)
	mux.HandleFunc("GET /users/{id}", getUser)
	mux.HandleFunc("DELETE /users/{id}", deleteUser)

	fmt.Println("Server listening on Port: 8080")
	http.ListenAndServe(":8080", mux)
}

// w sends header and response
// r contains all info of the request
func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User

	// creates a new decoder based on the body of the request and extrapolates the user data
	// that matches the struct
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if user.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	cacheMutex.Lock()
	userCache[len(userCache)+1] = user // race condition without mutex
	cacheMutex.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id")) // gets named value from req url param
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheMutex.RLock()
	user, ok := userCache[id]
	cacheMutex.RUnlock()

	if !ok {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	j, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheMutex.RLock()
	if _, ok := userCache[id]; !ok {
		http.Error(w, "user not found", http.StatusBadRequest)
		cacheMutex.RUnlock()
		return
	}
	cacheMutex.RUnlock()

	cacheMutex.Lock()
	delete(userCache, id)
	cacheMutex.Unlock()

	w.WriteHeader(http.StatusNoContent)
}
