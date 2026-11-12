package http

import (
	"encoding/json/v2"
	"net/http"
)

func writeJSONError(
	w http.ResponseWriter,
	httpStatus int,
	code string,
	message string,
) {
	resp := ErrorResponse{
		ErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	_, err = w.Write(data)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
