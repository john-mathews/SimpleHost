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

const templateFolderPath = "templates"

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./Database/simplehost.db")
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

func render(w http.ResponseWriter, _ *http.Request, name string, data any) {
	tmpl, err := template.New("").ParseFiles(filepath.Join(templateFolderPath, name), filepath.Join(templateFolderPath, "base.html"))
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}
	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func main() {
	db := initDB()
	defer db.Close()
	models.SetUserDB(db)
	models.SetFSDB(db)

	// Initialize virtual filesystem tables
	if err := models.InitVirtualFileSystemTables(); err != nil {
		log.Fatalf("Failed to initialize virtual filesystem tables: %v", err)
	}
	// Ensure root folder exists
	if err := models.EnsureRootFolder(); err != nil {
		log.Fatalf("Failed to create root folder: %v", err)
	}

	router := http.NewServeMux()

	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			render(w, r, "login.html", nil)
			return
		}
		// After successful login, redirect to /simplehost
		controllers.LoginHandler(w, r, func(w http.ResponseWriter, r *http.Request, name string, data any) {
			// If login is successful, LoginHandler should set the cookie and redirect
			// If not, render login.html with error
			if err, ok := data.(error); ok && err != nil {
				render(w, r, "login.html", map[string]any{"Error": err.Error()})
			} else {
				http.Redirect(w, r, "/simplehost", http.StatusSeeOther)
			}
		})
	})

	router.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			render(w, r, "register.html", nil)
			return
		}
		controllers.RegisterHandler(w, r, render)
	})

	router.HandleFunc("/success", controllers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		controllers.SuccessPageHandler(w, r, render)
	}))

	router.HandleFunc("/simplehost", controllers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		render(w, r, "simplehost.html", nil)
	}))

	router.HandleFunc("/success-message", controllers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		controllers.SuccessMessageHandler(w, r, render)
	}))

	router.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		controllers.LogoutHandler(w, r, render)
	})

	router.HandleFunc("/favicon.ico", http.NotFound)

	router.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		render(w, r, "404.html", nil)
	})

	// Serve static files from /static/ mapped to templates directory
	router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("templates"))))

	router.HandleFunc("/api/upload", controllers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		controllers.UploadHandler(w, r)
	}))

	// router.HandleFunc("/api/folder", controllers.AuthMiddleware(controllers.FolderChildrenAPIHandler))

	router.HandleFunc("/api/files-list", controllers.AuthMiddleware(controllers.FolderListPartialHandler))

	router.HandleFunc("/api/create-folder", controllers.AuthMiddleware(controllers.CreateFolderAPIHandler))

	// Download endpoint
	router.HandleFunc("/api/download", controllers.AuthMiddleware(controllers.DownloadHandler))

	// Breadcrumbs endpoint
	router.HandleFunc("/api/breadcrumbs", controllers.AuthMiddleware(controllers.BreadcrumbsHandler))

	// Catch-all handler for root and unknown routes
	// Keep this at the end to catch all unmatched routes
	router.HandleFunc("/", controllers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		render(w, r, "404.html", nil)
	}))

	log.Println("Server running on :8080")
	http.ListenAndServe(":8080", router)
}
