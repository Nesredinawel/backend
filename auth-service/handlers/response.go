package handlers

import (
	"auth-service/utils"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	utils.WriteJSON(w, status, data)
}

func writeError(w http.ResponseWriter, status int, code utils.ErrorCode, message, details string) {
	utils.WriteJSONError(w, &utils.ServiceError{
		Code:    code,
		Message: message,
		Details: details,
	}, status)
}

func writeServerError(w http.ResponseWriter, message string) {
	writeError(w, http.StatusInternalServerError, utils.ErrCodeServer, message, "")
}

func writeAuthError(w http.ResponseWriter, message string) {
	writeError(w, http.StatusUnauthorized, utils.ErrCodeAuth, message, "")
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, utils.ErrCodeBadRequest, message, "")
}

func writeConflict(w http.ResponseWriter, message string) {
	writeError(w, http.StatusConflict, utils.ErrCodeValidation, message, "")
}

func writeSuccess(w http.ResponseWriter, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["success"] = true
	writeJSON(w, http.StatusOK, data)
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
