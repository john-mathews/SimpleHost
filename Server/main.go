package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"simplehost-server/controllers"
	"simplehost-server/models"

	_ "github.com/mattn/go-sqlite3"
)

var templates *template.Template

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

func loadTemplates() {
	templates = template.Must(template.ParseGlob(filepath.Join("templates", "*.html")))
}

func render(w http.ResponseWriter, _ *http.Request, name string, data any) {
	t := templates.Lookup(name)
	if t == nil {
		http.Error(w, "Template not found: "+name, http.StatusInternalServerError)
		return
	}

	err := t.Execute(w, data)

	// err := templates.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	db := initDB()
	defer db.Close()
	models.SetDB(db)
	loadTemplates()
	router := http.NewServeMux()

	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			render(w, r, "login.html", nil)
			return
		}
		controllers.LoginHandler(w, r, render)
	})

	router.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			render(w, r, "register.html", nil)
			return
		}
		controllers.RegisterHandler(w, r, render)
	})

	router.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		if _, err := controllers.GetUserClaims(r); err != nil {
			w.WriteHeader(http.StatusNotFound)
			render(w, r, "404.html", nil)
			return
		}
		controllers.SuccessPageHandler(w, r, render)
	})

	// Success message route now uses the new handler
	router.HandleFunc("/success-message", func(w http.ResponseWriter, r *http.Request) {
		controllers.SuccessMessageHandler(w, r, render)
	})

	router.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		controllers.LogoutHandler(w, r, render)
	})

	router.HandleFunc("/favicon.ico", http.NotFound)

	router.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		render(w, r, "404.html", nil)
	})

	// Catch-all handler for root and unknown routes
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		render(w, r, "404.html", nil)
	})

	log.Println("Server running on :8080")
	http.ListenAndServe(":8080", router)
}
