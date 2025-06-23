package models

import (
	"database/sql"
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
}

type File struct {
	ID           string
	Name         string
	FolderID     string
	StoragePath  string
	OwnerID      string
	UploadedDate time.Time
	IsPrivate    bool
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
		files = append(files, file)
	}

	return folders, files, nil
}

// GetFileByID returns a File by its ID
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
