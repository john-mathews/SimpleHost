package controllers

import (
	"database/sql"
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

	// Map to keep track of created folders: relative path -> folder ID
	createdFolders := map[string]string{"": r.FormValue("folder_id")}
	if createdFolders[""] == "" {
		createdFolders[""] = "root"
	}

	for i, fileHeader := range files {
		src, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}
		defer src.Close()

		relativePaths := r.MultipartForm.Value["relative_path"]
		relativePath := fileHeader.Filename
		if len(relativePaths) > i {
			relativePath = relativePaths[i]
		}
		dir, _ := filepath.Split(relativePath)
		dir = strings.TrimSuffix(dir, string(os.PathSeparator))

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

		// Walk the directory structure and create folders as needed
		parentID := createdFolders[""]
		if dir != "" {
			parts := strings.Split(dir, string(os.PathSeparator))
			pathSoFar := ""
			for _, part := range parts {
				if part == "" {
					continue
				}
				pathSoFar = filepath.Join(pathSoFar, part)
				if _, exists := createdFolders[pathSoFar]; !exists {
					folder := models.Folder{
						ID:        part + "-" + fmt.Sprint(time.Now().UnixNano()),
						Name:      part,
						ParentID:  sql.NullString{String: parentID, Valid: parentID != ""},
						OwnerID:   ownerID,
						IsPrivate: false,
					}
					_ = models.InsertFolder(folder)
					createdFolders[pathSoFar] = folder.ID
					parentID = folder.ID
				} else {
					parentID = createdFolders[pathSoFar]
				}
			}
		}

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

		fileRecord := models.File{
			ID:           uniqueName, // Use uniqueName as ID for now, or use uuid if available
			Name:         uniqueName,
			FolderID:     parentID,
			StoragePath:  dstPath,
			OwnerID:      ownerID,
			UploadedDate: time.Now(),
			IsPrivate:    false,
		}
		_ = models.InsertFile(fileRecord)

		fmt.Fprintf(w, "Uploaded: %s\n", uniqueName)
	}
}
