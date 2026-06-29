package api

import (
	"encoding/json"
	"net/http"
)

// Error codes are the stable, machine-readable contract of the API: clients key
// off the code (not the message) to decide what to show. Messages are a
// human-readable hint (logs / fallback); they live here, next to their code, so
// they are not scattered as string literals across handlers.
const (
	codeInvalidRequest    = "invalid_request"
	codeInvalidEmail      = "invalid_email"
	codeInvalidGift       = "invalid_gift"
	codeInvalidRevealType = "invalid_reveal_type"
	codeInvalidPixelArt   = "invalid_pixel_art"
	codeInvalidID         = "invalid_id"
	codeUnauthorized      = "unauthorized"
	codeForbidden         = "forbidden"
	codeGiftNotFound      = "gift_not_found"
	codeInternalError     = "internal_error"
)

// errorMessages holds the default human-readable message for each error code.
var errorMessages = map[string]string{
	codeInvalidRequest:    "cuerpo JSON inválido",
	codeInvalidEmail:      "email inválido",
	codeInvalidGift:       "datos del regalo inválidos",
	codeInvalidRevealType: "tipo de revelación inválido",
	codeInvalidPixelArt:   "pixel art inválido",
	codeInvalidID:         "identificador inválido",
	codeUnauthorized:      "se requiere una sesión válida",
	codeForbidden:         "no tienes permiso sobre este regalo",
	codeGiftNotFound:      "regalo no encontrado",
	codeInternalError:     "error interno",
}

// errorEnvelope is the project's standard error response shape:
// {"error": {"code": "snake_case_code", "message": "texto legible"}}.
type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// respondError writes status with the standard error envelope, using the default
// message registered for code.
func respondError(w http.ResponseWriter, status int, code string) {
	respondJSON(w, status, errorEnvelope{Error: errorDetail{Code: code, Message: errorMessages[code]}})
}

// respondJSON writes v as a JSON body with the given status.
func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
