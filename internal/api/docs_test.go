package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func getPath(srv *Server, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func TestDocs_DisabledByDefault(t *testing.T) {
	srv := NewServer(nil, giftSessions(), nil, nil)

	require.Equal(t, http.StatusNotFound, getPath(srv, "/api/docs").Code)
	require.Equal(t, http.StatusNotFound, getPath(srv, "/api/docs/openapi.yaml").Code)
}

func TestDocs_EnabledServesSpecAndUI(t *testing.T) {
	srv := NewServer(nil, giftSessions(), nil, nil)
	srv.EnableDocs()

	spec := getPath(srv, "/api/docs/openapi.yaml")
	require.Equal(t, http.StatusOK, spec.Code)
	require.Contains(t, spec.Header().Get("Content-Type"), "application/yaml")
	require.Contains(t, spec.Body.String(), "openapi: 3.1.0")

	ui := getPath(srv, "/api/docs")
	require.Equal(t, http.StatusOK, ui.Code)
	require.Contains(t, ui.Header().Get("Content-Type"), "text/html")
	require.Contains(t, ui.Body.String(), "/api/docs/openapi.yaml")
}
