package models

import (
	"database/sql"
	"os"
	"time"
)

var db *sql.DB

func SetFSDB(database *sql.DB) {
	db = database
}

type Folder struct {
	ID        string
	Name      string
	ParentID  sql.NullString
	OwnerID   string
	IsPrivate bool
	CanDelete bool // Indicates if the user can delete this folder
}

type File struct {
	ID           string
	Name         string
	FolderID     string
	StoragePath  string
	OwnerID      string
	UploadedDate time.Time
	IsPrivate    bool
	CanDelete    bool // Indicates if the user can delete this file
}

func InitVirtualFileSystemTables() error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS folders (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id TEXT,
		owner_id TEXT NOT NULL,
		is_private BOOLEAN NOT NULL DEFAULT 0,
		FOREIGN KEY(parent_id) REFERENCES folders(id)
	);
	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		folder_id TEXT NOT NULL,
		storage_path TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		uploaded_date DATETIME NOT NULL,
		is_private BOOLEAN NOT NULL DEFAULT 0,
		FOREIGN KEY(folder_id) REFERENCES folders(id)
	);
	`)
	return err
}

// InsertFile inserts a new file record into the files table
func InsertFile(file File) error {
	_, err := db.Exec(`
		INSERT INTO files (id, name, folder_id, storage_path, owner_id, uploaded_date, is_private)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		file.ID,
		file.Name,
		file.FolderID,
		file.StoragePath,
		file.OwnerID,
		file.UploadedDate,
		file.IsPrivate,
	)
	return err
}

// InsertFolder inserts a new folder record into the folders table
func InsertFolder(folder Folder) error {
	_, err := db.Exec(`
		INSERT INTO folders (id, name, parent_id, owner_id, is_private)
		VALUES (?, ?, ?, ?, ?)
	`,
		folder.ID,
		folder.Name,
		folder.ParentID,
		folder.OwnerID,
		folder.IsPrivate,
	)
	return err
}

// EnsureRootFolder checks if the root folder exists and creates it if not
func EnsureRootFolder() error {
	const rootID = "root"
	var exists int
	err := db.QueryRow("SELECT COUNT(1) FROM folders WHERE id = ?", rootID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists == 0 {
		_, err := db.Exec("INSERT INTO folders (id, name, parent_id, owner_id, is_private) VALUES (?, ?, NULL, ?, 0)", rootID, "Root", "system")
		return err
	}
	return nil
}

// GetFolderPath returns all folders in the path from the given folder id up to root (always includes root)
func GetFolderPath(folderID string) ([]Folder, error) {
	var path []Folder
	currentID := folderID
	for currentID != "" {
		var f Folder
		var parentID sql.NullString
		err := db.QueryRow("SELECT id, name, parent_id, owner_id, is_private FROM folders WHERE id = ?", currentID).Scan(&f.ID, &f.Name, &parentID, &f.OwnerID, &f.IsPrivate)
		if err != nil {
			return nil, err
		}
		f.ParentID = parentID
		path = append([]Folder{f}, path...)
		if currentID == "root" {
			break
		}
		if !parentID.Valid {
			break
		}
		currentID = parentID.String
	}
	return path, nil
}

// GetFolderChildren returns all folders with parent_id = folderID and all files with folder_id = folderID
// Only returns private folders/files if ownerID matches the provided userID
func GetFolderChildren(folderID, userID string) ([]Folder, []File, error) {
	var folders []Folder
	var files []File

	// Get child folders
	rows, err := db.Query("SELECT id, name, parent_id, owner_id, is_private FROM folders WHERE parent_id = ?", folderID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f Folder
		var parentID sql.NullString
		if err := rows.Scan(&f.ID, &f.Name, &parentID, &f.OwnerID, &f.IsPrivate); err != nil {
			return nil, nil, err
		}
		f.ParentID = parentID
		if f.IsPrivate && f.OwnerID != userID {
			continue
		}
		f.CanDelete = (f.OwnerID == userID) // User can delete if they own the folder
		folders = append(folders, f)
	}

	// Get files in folder
	fileRows, err := db.Query("SELECT id, name, folder_id, storage_path, owner_id, uploaded_date, is_private FROM files WHERE folder_id = ?", folderID)
	if err != nil {
		return nil, nil, err
	}
	defer fileRows.Close()
	for fileRows.Next() {
		var file File
		var uploaded time.Time
		if err := fileRows.Scan(&file.ID, &file.Name, &file.FolderID, &file.StoragePath, &file.OwnerID, &uploaded, &file.IsPrivate); err != nil {
			return nil, nil, err
		}
		file.UploadedDate = uploaded
		if file.IsPrivate && file.OwnerID != userID {
			continue
		}
		file.CanDelete = (file.OwnerID == userID) // User can delete if they own the file
		files = append(files, file)
	}

	return folders, files, nil
}

func GetFileByID(fileID string) (*File, error) {
	var file File
	var uploaded time.Time
	err := db.QueryRow(`SELECT id, name, folder_id, storage_path, owner_id, uploaded_date, is_private FROM files WHERE id = ?`, fileID).
		Scan(&file.ID, &file.Name, &file.FolderID, &file.StoragePath, &file.OwnerID, &uploaded, &file.IsPrivate)
	if err != nil {
		return nil, err
	}
	file.UploadedDate = uploaded
	return &file, nil
}

// GetFileByFolderAndName returns a file in a given folder by its name, or nil if not found
func GetFileByFolderAndName(folderID, fileName string) (*File, error) {
	var file File
	var uploaded time.Time
	err := db.QueryRow(`SELECT id, name, folder_id, storage_path, owner_id, uploaded_date, is_private FROM files WHERE folder_id = ? AND name = ?`, folderID, fileName).
		Scan(&file.ID, &file.Name, &file.FolderID, &file.StoragePath, &file.OwnerID, &uploaded, &file.IsPrivate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	file.UploadedDate = uploaded
	return &file, nil
}

// DeleteFileByID deletes a file by its ID: removes the file from disk and deletes the DB record
func DeleteFileByID(fileID string, userID string) error {
	file, err := GetFileByID(fileID)
	if err != nil {
		return err
	}
	if file == nil {
		return nil // Already gone
	}
	if userID != file.OwnerID {
		return sql.ErrNoRows // Unauthorized access
	}

	// Remove file from disk
	if file.StoragePath != "" {
		if removeErr := os.Remove(file.StoragePath); removeErr != nil && !os.IsNotExist(removeErr) {
			return removeErr
		}
	}
	// Remove from DB
	_, err = db.Exec(`DELETE FROM files WHERE id = ?`, fileID)
	return err
}

// DeleteFolderByID deletes a folder from the DB (does not delete files)
func DeleteFolderByID(folderID string, userID string) error {
	folder, err := GetFolderByID(folderID)
	if err != nil {
		return err
	}
	if folder == nil {
		return nil // Already gone
	}
	if userID != folder.OwnerID {
		return sql.ErrNoRows // Unauthorized access
	}
	_, err = db.Exec(`DELETE FROM folders WHERE id = ?`, folderID)
	return err
}

// MoveFilesToParent moves all files in a folder to its parent folder
func MoveFilesToParent(folderID, parentID string) error {
	_, err := db.Exec(`UPDATE files SET folder_id = ? WHERE folder_id = ?`, parentID, folderID)
	return err
}

func MoveFoldersToParent(folderID, parentID string) error {
	_, err := db.Exec(`UPDATE folders SET parent_id = ? WHERE parent_id = ?`, parentID, folderID)
	return err
}

// GetAllFilesInFolderRecursive returns all files in a folder and its subfolders
func GetAllFilesInFolderRecursive(folderID string) ([]File, []Folder, error) {
	var files []File
	var folders []Folder
	var collect func(string) error
	collect = func(fid string) error {
		// Use empty userID to get all files/folders (no privacy filter)
		fldrs, fs, err := GetFolderChildren(fid, "")
		if err != nil {
			return err
		}
		files = append(files, fs...)
		folders = append(folders, fldrs...)
		for _, sub := range fldrs {
			if err := collect(sub.ID); err != nil {
				return err
			}
		}
		return nil
	}
	if err := collect(folderID); err != nil {
		return nil, nil, err
	}
	return files, folders, nil
}

// GetFolderByID returns a Folder by its ID
func GetFolderByID(folderID string) (*Folder, error) {
	var folder Folder
	var parentID sql.NullString
	err := db.QueryRow(`SELECT id, name, parent_id, owner_id, is_private FROM folders WHERE id = ?`, folderID).
		Scan(&folder.ID, &folder.Name, &parentID, &folder.OwnerID, &folder.IsPrivate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	folder.ParentID = parentID
	return &folder, nil
}

func ConvertNullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "" // Or handle the null case as needed, e.g., return a default value or an error
}
