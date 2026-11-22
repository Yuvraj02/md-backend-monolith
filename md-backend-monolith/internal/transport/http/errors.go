package httpserver

import (
	"errors"
	"net/http"

	"github.com/marketing-digest/pkg/errorsx"
)

func writeServiceError(w http.ResponseWriter, err error) {
	if err == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	code := http.StatusInternalServerError
	msg := "internal error"
	switch {
	case errors.Is(err, errorsx.ErrInvalidArgument):
		code, msg = http.StatusBadRequest, "invalid argument"
	case errors.Is(err, errorsx.ErrNotFound):
		code, msg = http.StatusNotFound, "not found"
	case errors.Is(err, errorsx.ErrUnauthenticated):
		code, msg = http.StatusUnauthorized, "unauthenticated"
	case errors.Is(err, errorsx.ErrPermissionDenied):
		code, msg = http.StatusForbidden, "permission denied"
	case errors.Is(err, errorsx.ErrConflict):
		code, msg = http.StatusConflict, "conflict"
	default:
		msg = "internal error"
	}
	writeJSON(w, code, map[string]string{"error": msg})
}