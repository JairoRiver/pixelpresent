package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/JairoRiver/pixelpresent/internal/domain"
)

// viewGift sends the public GET /g/{token} (no session).
func viewGift(t *testing.T, gift GiftService, token string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, giftSessions(), gift)
	req := httptest.NewRequest(http.MethodGet, "/g/"+token, nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func decodePublic(t *testing.T, rec *httptest.ResponseRecorder) publicGiftResponse {
	t.Helper()
	var resp publicGiftResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

func ptrTime(tm time.Time) *time.Time { return &tm }
func strPtr(s string) *string         { return &s }

func TestViewGift_Visible(t *testing.T) {
	fake := &fakeGiftService{viewOut: domain.Gift{
		Title:          "Para ti",
		Message:        "feliz día",
		PixelArt:       json.RawMessage(`{"width":1}`),
		RevealType:     "box",
		RevealConfig:   json.RawMessage(`{"speed":1}`),
		ViewToken:      "secret-token",
		RecipientEmail: strPtr("recipient@example.com"),
	}}

	rec := viewGift(t, fake, "secret-token")

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "secret-token", fake.gotToken)

	resp := decodePublic(t, rec)
	require.Equal(t, stateVisible, resp.State)
	require.NotNil(t, resp.Gift)
	require.Equal(t, "Para ti", resp.Gift.Title)
	require.Equal(t, "feliz día", resp.Gift.Message)

	// Private fields must never appear in the public view.
	body := rec.Body.String()
	require.NotContains(t, body, "secret-token")
	require.NotContains(t, body, "recipient@example.com")
	require.NotContains(t, body, "creator_id")
}

func TestViewGift_NotYetOpen(t *testing.T) {
	openAt := time.Now().Add(24 * time.Hour)
	fake := &fakeGiftService{viewOut: domain.Gift{
		Title:           "Sorpresa",
		ScheduledOpenAt: ptrTime(openAt),
	}}

	rec := viewGift(t, fake, "tok")

	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodePublic(t, rec)
	require.Equal(t, stateNotYetOpen, resp.State)
	require.Nil(t, resp.Gift, "no gift payload before it opens")
	require.NotNil(t, resp.ScheduledOpenAt)
	require.WithinDuration(t, openAt, *resp.ScheduledOpenAt, time.Second)
}

func TestViewGift_Expired(t *testing.T) {
	fake := &fakeGiftService{viewOut: domain.Gift{
		Title:     "Caducado",
		ExpiresAt: ptrTime(time.Now().Add(-time.Hour)),
	}}

	rec := viewGift(t, fake, "tok")

	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodePublic(t, rec)
	require.Equal(t, stateExpired, resp.State)
	require.Nil(t, resp.Gift)
}

func TestViewGift_AlreadyOpened(t *testing.T) {
	fake := &fakeGiftService{viewOut: domain.Gift{
		Title:      "Visto",
		SingleOpen: true,
		OpenedAt:   ptrTime(time.Now().Add(-time.Minute)),
	}}

	rec := viewGift(t, fake, "tok")

	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodePublic(t, rec)
	require.Equal(t, stateAlreadyOpened, resp.State)
	require.Nil(t, resp.Gift)
}

func TestViewGift_NotFound(t *testing.T) {
	fake := &fakeGiftService{viewErr: domain.ErrGiftNotFound}
	rec := viewGift(t, fake, uuid.NewString())

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, codeGiftNotFound, decodeErrorCode(t, rec))
}
