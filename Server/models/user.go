package models

import (
	"database/sql"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

func SetUserDB(db *sql.DB) {
	DB = db
}

type User struct {
	ID       string
	Username string
	Email    string
	Password string
}

func CreateUser(username, email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	id := uuid.New().String()
	_, err = DB.Exec("INSERT INTO users (id, username, email, password) VALUES (?, ?, ?, ?)", id, username, email, string(hash))
	if err != nil {
		return err
	}
	return nil
}

func VerifyUser(username, password string) bool {
	var hash string
	err := DB.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&hash)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GetUserByUsername(username string) (*User, error) {
	var user User
	err := DB.QueryRow("SELECT id, username, email, password FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
