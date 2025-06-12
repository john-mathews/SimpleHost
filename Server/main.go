package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"simplehost-server/controllers"
	"simplehost-server/models"

	_ "github.com/mattn/go-sqlite3"
)

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./Database/users.db")
	if err != nil {
		log.Fatal(err)
	}
	createTable := `CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func main() {
	db := initDB()
	defer db.Close()
	models.SetDB(db)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from Go server!")
	})

	http.HandleFunc("/api/success-message", controllers.SuccessMessageHandler)
	http.HandleFunc("/api/login", controllers.LoginHandler)
	http.HandleFunc("/api/register", controllers.RegisterHandler)

	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", nil)
}
