package controllers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"
)

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		return false
	}
	_, err = ValidateJWT(cookie.Value)
	return err == nil
}

func setCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "http://localhost:5173" || origin == "http://host.docker.internal:5173" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func SuccessMessageHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w, r)

	if !isAuthenticated(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	messages := []string{
		"Success! Everything went perfectly!",
		"Great job! Your request was successful!",
		"Hooray! Operation completed with a smile!",
		"Awesome! You did it!",
		"Fantastic! All systems go!",
	}
	rand.Seed(time.Now().UnixNano())
	msg := messages[rand.Intn(len(messages))]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": msg})
}
