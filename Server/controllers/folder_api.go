package controllers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"simplehost-server/models"
	"strings"

	"simplehost-server/shared" // Import shared for TemplatesFS

	"github.com/google/uuid"
)

type NullString = sql.NullString

// FolderChildrenAPIHandler returns all child folders and files for a given folder id (or root if not provided)
func FolderChildrenAPIHandler(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folder")
	if folderID == "" {
		folderID = "root"
	}
	claims, _ := r.Context().Value("claims").(map[string]any)
	userID := ""
	if claims != nil {
		if id, ok := claims["userId"].(string); ok {
			userID = id
		}
	}
	folders, files, err := models.GetFolderChildren(folderID, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"folders": folders,
		"files":   files,
	})
}

// FolderListPartialHandler renders the files/folders list as HTML for htmx
func FolderListPartialHandler(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folderId")
	if folderID == "" {
		folderID = "root"
	}
	claims, _ := r.Context().Value("claims").(map[string]any)
	userID := ""
	if claims != nil {
		if id, ok := claims["userId"].(string); ok {
			userID = id
		}
	}
	folders, files, err := models.GetFolderChildren(folderID, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("<div class='error'>Error loading files</div>"))
		return
	}
	tmpl, err := template.ParseFS(shared.TemplatesFS, "templates/files_list_partial.html", "templates/folder_item_partial.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("<div class='error'>Template error</div>"))
		return
	}
	data := map[string]any{
		"Folders":         folders,
		"Files":           files,
		"CurrentFolderID": folderID,
	}
	tmpl.Execute(w, data)
}

// CreateFolderAPIHandler creates a new folder as a child of the given parent folder
func CreateFolderAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	parentID := r.FormValue("parent_id")
	if parentID == "" {
		parentID = "root"
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Folder name required"})
		return
	}
	claims, _ := r.Context().Value("claims").(map[string]any)
	ownerID := ""
	if claims != nil {
		if id, ok := claims["userId"].(string); ok {
			ownerID = id
		}
	}
	folder := models.Folder{
		ID:        uuid.New().String(),
		Name:      name,
		ParentID:  NullString{String: parentID, Valid: parentID != ""},
		OwnerID:   ownerID,
		IsPrivate: false,
	}
	err := models.InsertFolder(folder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFS(shared.TemplatesFS, "templates/folder_item_partial.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("<div class='error'>Template error</div>"))
		return
	}
	tmpl.Execute(w, folder)
}
