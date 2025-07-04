package main

import (
	"database/sql"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"simplehost-server/controllers"
	"simplehost-server/models"
	"simplehost-server/shared"

	_ "modernc.org/sqlite"
)

const templateFolderPath = "templates"

func initDB() *sql.DB {
	dbDir := "./Database"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}
	db, err := sql.Open("sqlite", dbDir+"/simplehost.db")
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

func getUploaderJSVersion() string {
	// Try to get the mod time of uploader.js from the embedded FS
	info, err := fs.Stat(shared.StaticFS, "templates/uploader.js")
	if err == nil {
		return info.ModTime().Format("20060102150405") // yyyymmddhhmmss
	}
	return time.Now().Format("20060102150405") // fallback: current time
}

func mergeData(data any, extra map[string]any) map[string]any {
	m, ok := data.(map[string]any)
	if !ok {
		m = map[string]any{"Data": data}
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func render(w http.ResponseWriter, _ *http.Request, name string, data any) {
	tmpl, err := template.New("").ParseFS(shared.TemplatesFS, templateFolderPath+"/"+name, templateFolderPath+"/base.html")
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}
	version := getUploaderJSVersion()
	merged := mergeData(data, map[string]any{"UploaderJSVersion": version})
	err = tmpl.ExecuteTemplate(w, "base", merged)
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

	// Serve static files (uploader.js) from embedded FS
	staticFiles, _ := fs.Sub(shared.StaticFS, "templates")
	router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))

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

	router.HandleFunc("/api/file/", controllers.AuthMiddleware(controllers.DeleteFileHandler))
	router.HandleFunc("/api/folder/delete", controllers.AuthMiddleware(controllers.DeleteFolderHandler))

	log.Println("Server running on :8080")
	http.ListenAndServe(":8080", router)
}
