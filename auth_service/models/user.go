package models

import "github.com/golang-jwt/jwt/v4"

// Claims struct with username and registered claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}
