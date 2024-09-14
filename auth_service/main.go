package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/samims/todo-auth/db"
	"github.com/samims/todo-auth/handlers"

	_ "github.com/lib/pq"
)

func main() {
	// initialize db
	db.InitDB()
	r := mux.NewRouter()

	r.HandleFunc("/ping", handlers.Ping)
	// public routes
	r.HandleFunc("/register", handlers.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", handlers.Login).Methods(http.MethodPost)
	r.HandleFunc("/validate", handlers.ValidateToken)

	log.Println("auth service starting on port 8000")

	log.Fatal(http.ListenAndServe(":8000", r))
}
