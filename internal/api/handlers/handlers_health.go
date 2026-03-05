package handlers

import (
	"net/http"

	"MarketNews/internal/api/helpers"
)

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		helpers.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
