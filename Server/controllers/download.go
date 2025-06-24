package controllers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"simplehost-server/models"
)

// DownloadHandler serves a file by fileId as a download
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("fileId")
	if fileID == "" {
		http.Error(w, "Missing fileId", http.StatusBadRequest)
		return
	}
	claims, _ := r.Context().Value("claims").(map[string]any)
	userID := ""
	if claims != nil {
		if id, ok := claims["userId"].(string); ok {
			userID = id
		}
	}
	file, err := models.GetFileByID(fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if file.IsPrivate && file.OwnerID != userID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}
	f, err := os.Open(filepath.Clean(file.StoragePath))
	if err != nil {
		http.Error(w, "Could not open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Expires", "0")
	io.Copy(w, f)
}

func PreviewHandler(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("fileId")
	if fileID == "" {
		http.Error(w, "Missing fileId", http.StatusBadRequest)
		return
	}
	claims, _ := r.Context().Value("claims").(map[string]any)
	userID := ""
	if claims != nil {
		if id, ok := claims["userId"].(string); ok {
			userID = id
		}
	}
	file, err := models.GetFileByID(fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if file.IsPrivate && file.OwnerID != userID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}
	f, err := os.Open(filepath.Clean(file.StoragePath))
	if err != nil {
		http.Error(w, "Could not open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, r, file.Name, file.UploadedDate, f)
}
