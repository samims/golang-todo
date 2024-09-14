package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/samims/todo-service/constants"
	"github.com/samims/todo-service/middlewares"
)

var db *sql.DB

// initDB initializes the database connection using the provided
// PostgreSQL connection details. It logs a fatal error if the
// connection cannot be established or if the ping to the database fails.
func initDB() {

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s "+"password=%s dbname=%s sslmode=disable",
		dbHost,
		dbPort,
		dbUser,
		dbPassword,
		dbName,
	)

	var err error

	db, err = sql.Open("postgres", psqlInfo)

	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}

	err = db.Ping()

	if err != nil {
		// Log a fatal error if the ping fails
		log.Fatalf("Error pinging database: %v", err)

	}
	// Log a success message if the connection is established
	log.Println("Successfully connected to the database")

	// Create table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS  todos (
	    id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		completed BOOLEAN NOT NULL DEFAULT FALSE,
		user_id INT
	);
	`
	// createTableSQL := `DROP table todos;`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error creating todos table: %v", err)
	}

}

// handlers

func getTodos(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query("select id, title, completed from todos")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}(rows)

	var todos []Todo

	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}
	err = json.NewEncoder(w).Encode(todos)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getTodoByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)

	id, _ := strconv.Atoi(params["id"])

	var todo Todo
	err := db.QueryRow("SELECT id, title, completed FROM todos WHERE id=$1", id).Scan(
		&todo.ID, &todo.Title, &todo.Completed,
	)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}
	err = json.NewEncoder(w).Encode(todo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func createTodo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var todo Todo

	// Decode the body into the to-do struct
	err := json.NewDecoder(r.Body).Decode(&todo)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = db.QueryRow(
		"INSERT INTO todos (title, completed) VALUES ($1, $2) RETURNING id",
		todo.Title, todo.Completed,
	).Scan(&todo.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(todo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func updateTodo(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)

	todID, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "id missing in url", http.StatusUnprocessableEntity)
	}

	var todo Todo
	err = json.NewDecoder(r.Body).Decode(&todo)

	if err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("userID").(int)

	result, err := db.Exec(`
		UPDATE todos set title=$1, completed=$2
		WHERE id=$3 AND user_id=$4`,
		todo.Title, todo.Completed, todID, userID,
	)

	rowsAffected, _ := result.RowsAffected()

	if err != nil || rowsAffected == 0 {
		http.Error(w, "failed to update todo or unauthorized", http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(map[string]string{"message": "To-do update successful!"})
	if err != nil {
		return
	}

}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	todoID, _ := strconv.Atoi(params["id"])

	userID := r.Context().Value(constants.USERNAME).(string)

	_, err := db.Exec("DELETE FROM todos WHERE id=$1 AND user_id=$2", todoID, userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(map[string]string{"result": "success"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	initDB()

	// close DB connection when everything is done
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("Error closing database connection: %v", err)
		}
	}(db)

	// setup router
	r := mux.NewRouter()

	// public routes
	publicRoutes := r.PathPrefix("/").Subrouter()

	publicRoutes.HandleFunc("/todos", getTodos).Methods(http.MethodGet)

	// authenticated routes
	authenticatedRoutes := r.PathPrefix("/").Subrouter()
	// add middleware
	authenticatedRoutes.Use(middlewares.ValidateTokenMiddleware)

	authenticatedRoutes.HandleFunc("/todos/{id}", getTodoByID).Methods(http.MethodGet)
	authenticatedRoutes.HandleFunc("/todos", createTodo).Methods(http.MethodPost)
	authenticatedRoutes.HandleFunc("/todos/{id}", updateTodo).Methods(http.MethodPut)
	authenticatedRoutes.HandleFunc("/todos/{id}", deleteTodo).Methods(http.MethodDelete)

	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
