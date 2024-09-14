package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/samims/todo-auth/db"
	"github.com/samims/todo-auth/models"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "pong")
}

func Register(w http.ResponseWriter, r *http.Request) {
	var user models.User

	// Decode request body to user struct
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Hash the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Insert user into the database
	err = db.Client.QueryRow("INSERT INTO users(username, password) VALUES($1, $2) RETURNING id", user.Username, string(hashedPassword)).Scan(&user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var user models.User
	defer r.Body.Close()
	// Decode request body to user struct
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var hashedPassword string

	// Query the database for the user's credentials
	err = db.Client.QueryRow("SELECT id, password FROM users WHERE username=$1", user.Username).Scan(&user.ID, &hashedPassword)
	if err == sql.ErrNoRows {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Compare the hashed password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate a JWT token
	expirationTime := time.Now().Add(24 * 5 * time.Hour)
	claims := &models.Claims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	// create the token with the specified claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// sign the token using secret key
	tokenStr, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"token":      tokenStr,
		"expiringAt": expirationTime,
	})

}

func ValidateToken(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request struct {
		Token string `json:"token"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	claims, err := validateTokenString(request.Token)

	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"valid":    true,
		"username": claims.Username,
	})

}

func validateTokenString(tokenStr string) (*models.Claims, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// getUserIDHandler handles fetching the userID from username
func GetUserIDHandler(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var userID int

	err := db.Client.QueryRow("SELECT id FROM users where username=$1", requestData.Username).Scan(&userID)

	if err == sql.ErrNoRows {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	resp := map[string]int{"user_id": userID}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

}
