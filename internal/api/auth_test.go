package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeRequester is an in-memory MagicLinkRequester recording its last call.
type fakeRequester struct {
	calls    int
	gotEmail string
	err      error
}

func (f *fakeRequester) RequestMagicLink(_ context.Context, email string) error {
	f.calls++
	f.gotEmail = email
	return f.err
}

// doRequest sends body to POST /auth/magic-link through the real chi router.
func doRequest(t *testing.T, svc MagicLinkRequester, body string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(svc)
	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func decodeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env errorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env.Error.Code
}

func TestRequestMagicLink_Accepted(t *testing.T) {
	fake := &fakeRequester{}
	rec := doRequest(t, fake, `{"email":"  Alice@Example.com  "}`)

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, 1, fake.calls)
	require.Equal(t, "Alice@Example.com", fake.gotEmail, "email should be trimmed before the service")
}

func TestRequestMagicLink_BadJSON(t *testing.T) {
	fake := &fakeRequester{}
	rec := doRequest(t, fake, `{not json`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidRequest, decodeErrorCode(t, rec))
	require.Zero(t, fake.calls, "the service must not be called on a malformed body")
}

func TestRequestMagicLink_InvalidEmail(t *testing.T) {
	fake := &fakeRequester{}
	rec := doRequest(t, fake, `{"email":"not-an-email"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidEmail, decodeErrorCode(t, rec))
	require.Zero(t, fake.calls, "the service must not be called on an invalid email")
}

func TestRequestMagicLink_InternalError(t *testing.T) {
	fake := &fakeRequester{err: errors.New("smtp down")}
	rec := doRequest(t, fake, `{"email":"bob@example.com"}`)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, codeInternalError, decodeErrorCode(t, rec))
}

// TestRequestMagicLink_DoesNotLeakExistence documents the anti-enumeration
// contract: any valid email yields the same 202, so a caller cannot tell a
// registered address from an unregistered one.
func TestRequestMagicLink_DoesNotLeakExistence(t *testing.T) {
	fake := &fakeRequester{}

	recNew := doRequest(t, fake, `{"email":"new@example.com"}`)
	recExisting := doRequest(t, fake, `{"email":"existing@example.com"}`)

	require.Equal(t, http.StatusAccepted, recNew.Code)
	require.Equal(t, http.StatusAccepted, recExisting.Code)
	require.Empty(t, recNew.Body.String())
	require.Empty(t, recExisting.Body.String())
}
