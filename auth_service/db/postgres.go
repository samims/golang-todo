package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

var Client *sql.DB

func InitDB() *sql.DB {
	if Client != nil {
		return Client
	}
	dbHost := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost,
		port,
		dbUser,
		dbPassword,
		dbName,
	)

	var err error
	Client, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)

	}

	err = Client.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("db client initialized ....")
	return Client
}
