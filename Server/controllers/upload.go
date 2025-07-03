package controllers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// UploadHandler handles file uploads (chunked and non-chunked)
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

	// Chunked upload: check for chunk fields
	chunkFile, _, chunkErr := r.FormFile("chunk")
	fileName := r.FormValue("file_name")
	uploadId := r.FormValue("upload_id")
	chunkIdx := r.FormValue("chunk_index")
	totalChunks := r.FormValue("total_chunks")
	folderId := r.FormValue("folder_id")

	if chunkErr == nil && fileName != "" && uploadId != "" && chunkIdx != "" && totalChunks != "" {
		// Handle chunked upload
		defer chunkFile.Close()
		tmpDir := filepath.Join(uploadDir, ".chunks", uploadId)
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			http.Error(w, "Could not create chunk dir", http.StatusInternalServerError)
			return
		}
		chunkPath := filepath.Join(tmpDir, chunkIdx)
		out, err := os.Create(chunkPath)
		if err != nil {
			http.Error(w, "Could not create chunk file", http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(out, chunkFile); err != nil {
			out.Close()
			http.Error(w, "Could not write chunk", http.StatusInternalServerError)
			return
		}
		out.Close()
		// If last chunk, assemble
		if chunkIdx == fmt.Sprintf("%d", atoi(totalChunks)-1) {
			finalPath := filepath.Join(uploadDir, getUniqueFilename(uploadDir, fileName))
			finalOut, err := os.Create(finalPath)
			if err != nil {
				http.Error(w, "Could not create final file", http.StatusInternalServerError)
				return
			}
			for i := 0; i < atoi(totalChunks); i++ {
				chunkPath := filepath.Join(tmpDir, fmt.Sprintf("%d", i))
				in, err := os.Open(chunkPath)
				if err != nil {
					finalOut.Close()
					http.Error(w, "Missing chunk", http.StatusInternalServerError)
					return
				}
				if _, err := io.Copy(finalOut, in); err != nil {
					in.Close()
					finalOut.Close()
					http.Error(w, "Error assembling file", http.StatusInternalServerError)
					return
				}
				in.Close()
			}
			finalOut.Close()
			os.RemoveAll(tmpDir)
			// Insert file record
			claims, _ := r.Context().Value("claims").(map[string]any)
			ownerID := ""
			if claims != nil {
				if userId, ok := claims["userId"].(string); ok {
					ownerID = userId
				}
			}
			if ownerID == "" {
				http.Error(w, "Unauthorized: No valid user found", http.StatusUnauthorized)
				return
			}
			fileRecord := models.File{
				ID:           fileName + "-" + uploadId,
				Name:         fileName,
				FolderID:     folderId,
				StoragePath:  finalPath,
				OwnerID:      ownerID,
				UploadedDate: time.Now(),
				IsPrivate:    false,
			}
			_ = models.InsertFile(fileRecord)
			fmt.Fprintf(w, "Uploaded: %s\n", fileName)
			return
		}
		fmt.Fprintf(w, "Chunk %s uploaded\n", chunkIdx)
		return
	}

	// Fallback: non-chunked upload (legacy, for small files)
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}

	createdFolders := map[string]string{"": folderId}
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
