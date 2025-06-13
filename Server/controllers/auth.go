package controllers

import (
	"html/template"
	"net/http"
	"os"
	"time"

	"simplehost-server/models"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte(getJWTSecret())

func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "dev_secret_key_change_me" // fallback for dev
	}
	return secret
}

func GenerateJWT(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenMalformed
	}
	return claims, nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request, render func(http.ResponseWriter, *http.Request, string, any)) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if models.VerifyUser(username, password) {
		token, err := GenerateJWT(username)
		if err != nil {
			render(w, r, "login.html", map[string]any{"Error": "Server error"})
			return
		}
		cookie := &http.Cookie{
			Name:     "jwt",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/success", http.StatusSeeOther)
	} else {
		render(w, r, "login.html", map[string]any{"Error": "Invalid username or password"})
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request, render func(http.ResponseWriter, *http.Request, string, any)) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")
	retype := r.FormValue("retype")
	if password != retype {
		render(w, r, "register.html", map[string]any{"Error": "Passwords do not match"})
		return
	}
	if !validatePassword(password) {
		render(w, r, "register.html", map[string]any{"Error": "Password must be at least 8 characters and include uppercase, lowercase, number, and special character."})
		return
	}
	if err := models.CreateUser(username, email, password); err != nil {
		render(w, r, "register.html", map[string]any{"Error": "Username or email already exists."})
		return
	}
	render(w, r, "login.html", map[string]any{"Error": template.HTML(`<span class='success'>Account created! Please log in.</span>`)})
}

func validatePassword(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false
	for _, c := range pw {
		switch {
		case 'A' <= c && c <= 'Z':
			hasUpper = true
		case 'a' <= c && c <= 'z':
			hasLower = true
		case '0' <= c && c <= '9':
			hasNumber = true
		case (c >= 33 && c <= 47) || (c >= 58 && c <= 64) || (c >= 91 && c <= 96) || (c >= 123 && c <= 126):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func SuccessPageHandler(w http.ResponseWriter, r *http.Request, render func(http.ResponseWriter, *http.Request, string, any)) {
	claims, err := getUserClaims(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]any{"Username": claims["username"], "Message": ""}
	render(w, r, "success.html", data)
}

func getUserClaimsUsername(r *http.Request) string {
	claims, err := getUserClaims(r)
	if err != nil {
		return ""
	}
	if username, ok := claims["username"].(string); ok {
		return username
	}
	return ""
}

func LogoutHandler(w http.ResponseWriter, r *http.Request, render func(http.ResponseWriter, *http.Request, string, any)) {
	cookie := &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	render(w, r, "login.html", map[string]any{"Error": template.HTML(`<span class='success'>Logged out.</span>`)})
}

// Export getUserClaims so it can be used in main.go
func GetUserClaims(r *http.Request) (map[string]any, error) {
	return getUserClaims(r)
}

func getUserClaims(r *http.Request) (map[string]any, error) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		return nil, err
	}
	claims, err := ValidateJWT(cookie.Value)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
