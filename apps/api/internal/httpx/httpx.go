// Package httpx holds the small JSON/HTTP helpers shared by every module.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ErrorBody struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func Error(w http.ResponseWriter, status int, code, msg string) {
	JSON(w, status, ErrorBody{Error: code, Message: msg})
}

// Invalid reports field-level validation errors (422).
func Invalid(w http.ResponseWriter, fields map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, ErrorBody{Error: "validation_failed", Message: "Periksa kembali isian Anda.", Fields: fields})
}

func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "not_found", "Data tidak ditemukan.")
}

func Forbidden(w http.ResponseWriter) { Error(w, http.StatusForbidden, "forbidden", "Akses ditolak.") }

// Internal logs the real error and returns an opaque 500.
func Internal(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "internal error", "err", err, "path", r.URL.Path)
	Error(w, http.StatusInternalServerError, "internal", "Terjadi kesalahan. Silakan coba lagi.")
}

const maxBody = 1 << 20

// Decode parses a JSON body strictly (unknown fields rejected, 1 MiB cap).
func Decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		msg := "Format permintaan tidak valid."
		var syn *json.SyntaxError
		if errors.As(err, &syn) || errors.Is(err, io.EOF) {
			msg = "Body JSON tidak valid."
		} else if strings.HasPrefix(err.Error(), "json: unknown field") {
			msg = err.Error()
		}
		Error(w, http.StatusBadRequest, "bad_request", msg)
		return false
	}
	return true
}

// UUIDParam reads a uuid path parameter, answering 404 when malformed.
func UUIDParam(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		NotFound(w)
		return uuid.Nil, false
	}
	return id, true
}

func IntParam(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	n, err := strconv.Atoi(chi.URLParam(r, name))
	if err != nil {
		NotFound(w)
		return 0, false
	}
	return n, true
}

// Int64Param reads a bigint path parameter, answering 404 when malformed.
func Int64Param(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	n, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		NotFound(w)
		return 0, false
	}
	return n, true
}

func QueryInt(r *http.Request, name string, def int) int {
	if n, err := strconv.Atoi(r.URL.Query().Get(name)); err == nil {
		return n
	}
	return def
}
