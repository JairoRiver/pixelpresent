package api

import (
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestStatic_NotServedByDefault(t *testing.T) {
	srv := NewServer(nil, giftSessions(), nil, nil)

	require.Equal(t, http.StatusNotFound, getPath(srv, "/").Code)
}

func TestStatic_ServesEmbeddedSite(t *testing.T) {
	srv := NewServer(nil, giftSessions(), nil, nil)
	srv.ServeStatic(fstest.MapFS{
		"index.html":     {Data: []byte("<h1>home</h1>")},
		"_astro/app.css": {Data: []byte(".x{}")},
	})

	home := getPath(srv, "/")
	require.Equal(t, http.StatusOK, home.Code)
	require.Contains(t, home.Body.String(), "home")

	css := getPath(srv, "/_astro/app.css")
	require.Equal(t, http.StatusOK, css.Code)
	require.Contains(t, css.Header().Get("Content-Type"), "css")

	// A path with no matching file 404s (no SPA index fallback).
	require.Equal(t, http.StatusNotFound, getPath(srv, "/missing.txt").Code)
}

func TestReveal_ServesPageForTokenizedURL(t *testing.T) {
	srv := NewServer(nil, giftSessions(), nil, nil)
	srv.ServeStatic(fstest.MapFS{
		"index.html":   {Data: []byte("<h1>home</h1>")},
		"g/index.html": {Data: []byte("<h1>reveal</h1>")},
	})

	// The tokenized public URL has no matching file, but the reveal route serves
	// the reveal document rather than 404ing (the token is read client-side).
	res := getPath(srv, "/g/some-view-token")
	require.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.Body.String(), "reveal")
}
