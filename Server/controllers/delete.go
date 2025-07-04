package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"simplehost-server/models"
)

// File delete endpoint: DELETE /api/file/{id}
func DeleteFileHandler(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/api/file/")
	userID := GetUserIDFromRequest(r)
	file, err := models.GetFileByID(fileID)
	if err != nil || file == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if file.OwnerID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err := models.DeleteFileByID(fileID, userID); err != nil {
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// Folder delete endpoint: POST /api/folder/delete
// Accepts JSON: { "folder_id": "...", "mode": "folder"|"all" }
func DeleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	type reqBody struct {
		FolderID string `json:"folder_id"`
		Mode     string `json:"mode"` // "folder" or "all"
	}
	var req reqBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	userID := GetUserIDFromRequest(r)
	folder, err := models.GetFolderByID(req.FolderID)
	if err != nil || folder == nil {
		http.Error(w, "Folder not found", http.StatusNotFound)
		return
	}
	if folder.OwnerID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch req.Mode {
	case "folder":
		// Move files to parent, then delete folder
		var parentID = models.ConvertNullStringToString(folder.ParentID)
		if parentID == "" {
			http.Error(w, "Cannot delete root folder", http.StatusBadRequest)
			return
		}
		if err := models.MoveFilesToParent(req.FolderID, parentID); err != nil {
			http.Error(w, "Failed to move files", http.StatusInternalServerError)
			return
		}

		if err := models.MoveFoldersToParent(req.FolderID, parentID); err != nil {
			http.Error(w, "Failed to move subfolders", http.StatusInternalServerError)
			return
		}

		if err := models.DeleteFolderByID(req.FolderID, userID); err != nil {
			http.Error(w, "Failed to delete folder", http.StatusInternalServerError)
			return
		}
	case "all":
		// Delete all files recursively, then delete folder
		files, folders, err := models.GetAllFilesInFolderRecursive(req.FolderID)
		if err != nil {
			http.Error(w, "Failed to get files", http.StatusInternalServerError)
			return
		}
		for _, f := range files {
			err = models.DeleteFileByID(f.ID, userID)
			if err != nil {
				http.Error(w, "Failed to delete file", http.StatusInternalServerError)
				return
			}
		}
		for _, f := range folders {
			err = models.DeleteFolderByID(f.ID, userID)
			if err != nil {
				http.Error(w, "Failed to delete folder", http.StatusInternalServerError)
				return
			}
		}

		if err := models.DeleteFolderByID(req.FolderID, userID); err != nil {
			http.Error(w, "Failed to delete folder", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Invalid mode", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
