package twitchgo

import "net/http"

func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
