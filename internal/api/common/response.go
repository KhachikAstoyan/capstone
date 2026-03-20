package common

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

func RespondError(w http.ResponseWriter, status int, err error, message string) {
	response := ErrorResponse{
		Error:   err.Error(),
		Message: message,
	}
	RespondJSON(w, status, response)
}

func RespondSimpleError(w http.ResponseWriter, status int, message string) {
	response := ErrorResponse{
		Error: message,
	}
	RespondJSON(w, status, response)
}
