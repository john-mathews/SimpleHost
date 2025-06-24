package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"simplehost-server/models"
)

const uploadDir = "simplehostdata"

// getUniqueFilename returns a unique filename in the directory by appending (n) if needed
func getUniqueFilename(dir, filename string) string {
	base := filename
	ext := ""
	if dot := strings.LastIndex(filename, "."); dot != -1 {
		base = filename[:dot]
		ext = filename[dot:]
	}
	unique := filename
	count := 1
	for {
		if _, err := os.Stat(filepath.Join(dir, unique)); os.IsNotExist(err) {
			return unique
		}
		unique = fmt.Sprintf("%s(%d)%s", base, count, ext)
		count++
	}
}

// UploadHandler handles file uploads and saves them to the simplehostdata directory
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		http.Error(w, "Could not create upload directory", http.StatusInternalServerError)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}

	for _, fileHeader := range files {
		src, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}
		defer src.Close()

		uniqueName := getUniqueFilename(uploadDir, fileHeader.Filename)
		dstPath := filepath.Join(uploadDir, uniqueName)
		dst, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			http.Error(w, "Error writing file", http.StatusInternalServerError)
			return
		}

		claims, _ := r.Context().Value("claims").(map[string]any)
		ownerID := ""
		if claims != nil {
			if userId, ok := claims["userId"].(string); ok {
				ownerID = userId
			}
		}
		if ownerID == "" {
			//logout because we don't have a valid user
			http.Error(w, "Unauthorized: No valid user found", http.StatusUnauthorized)
			return
		}
		folderID := r.FormValue("folder_id")
		if folderID == "" {
			folderID = "root"
		}
		fileRecord := models.File{
			ID:           uniqueName, // Use uniqueName as ID for now, or use uuid if available
			Name:         uniqueName,
			FolderID:     folderID,
			StoragePath:  dstPath,
			OwnerID:      ownerID,
			UploadedDate: time.Now(),
			IsPrivate:    false,
		}
		_ = models.InsertFile(fileRecord)

		fmt.Fprintf(w, "Uploaded: %s\n", uniqueName)
	}
}
