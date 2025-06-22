package controllers

import (
	"math/rand"
	"net/http"
)

func SuccessMessageHandler(w http.ResponseWriter, r *http.Request, _ func(http.ResponseWriter, *http.Request, string, any)) {
	if _, err := getUserClaims(r); err != nil {
		w.WriteHeader(http.StatusForbidden)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	messages := []string{
		"Success! Everything went perfectly!",
		"Great job! Your request was successful!",
		"Hooray! Operation completed with a smile!",
		"Awesome! You did it!",
		"Fantastic! All systems go!",
	}
	msg := messages[rand.Intn(len(messages))]
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(msg))
}
