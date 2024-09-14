package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/samims/todo-service/constants"
)

func ValidateTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractTokenFromHeader(r)

		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		authServiceURL := os.Getenv("AUTH_VALIDATION_URL") // Update with your auth service URL
		reqBody, err := json.Marshal(map[string]string{
			"token": token,
		})

		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		req, err := http.NewRequest("POST", authServiceURL, bytes.NewBuffer(reqBody))
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var response map[string]any
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		type usernameCtx string

		usernameCtxKey := usernameCtx(constants.USERNAME)

		// Store the username from the token in the request context
		ctx := context.WithValue(r.Context(), usernameCtxKey, response[constants.USERNAME])
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	// return the second part
	return parts[1]
}
