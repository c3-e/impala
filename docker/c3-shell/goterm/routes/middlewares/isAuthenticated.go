package middlewares

import (
	"app"
	"net/http"
)

func IsAuthenticated(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, err := app.Store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, ok := session.Values["profile"]; !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
