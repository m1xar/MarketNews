package handlers

import (
	"net/http"

	"MarketNews/internal/api/helpers"
)

func HandleSwaggerRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		helpers.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	http.Redirect(w, r, "/swagger/index.html", http.StatusTemporaryRedirect)
}
