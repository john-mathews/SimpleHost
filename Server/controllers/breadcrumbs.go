package controllers

import (
	"encoding/json"
	"net/http"
	"simplehost-server/models"
)

// BreadcrumbsHandler returns the folder path from root to the current folder as JSON
func BreadcrumbsHandler(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folderId")
	if folderID == "" {
		folderID = "root"
	}
	path, err := models.GetFolderPath(folderID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(path)
}
