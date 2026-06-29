package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/JairoRiver/pixelpresent/internal/auth"
	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/gifts"
)

// fakeGiftService records the input it was called with and returns a scripted
// result.
type fakeGiftService struct {
	calls    int
	gotInput gifts.CreateInput
	out      domain.Gift
	err      error
}

func (f *fakeGiftService) Create(_ context.Context, in gifts.CreateInput) (domain.Gift, error) {
	f.calls++
	f.gotInput = in
	return f.out, f.err
}

const giftTestSecret = "gift-secret"

func giftSessions() *auth.SessionManager {
	return auth.NewSessionManager(giftTestSecret, false, time.Hour)
}

// postGift sends body to POST /gifts with a valid session cookie for userID.
func postGift(t *testing.T, gift GiftService, sessions *auth.SessionManager, userID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, sessions, gift)

	issue := httptest.NewRecorder()
	sessions.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodPost, "/gifts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(issue.Result().Cookies()[0])

	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

const validGiftBody = `{
	"title": "Feliz cumple",
	"message": "para ti",
	"pixel_art": {"width":1,"height":1,"palette":["#000000"],"pixels":[0]},
	"reveal_type": "box",
	"single_open": true
}`

func TestCreateGift_RequiresSession(t *testing.T) {
	fake := &fakeGiftService{}
	srv := NewServer(nil, giftSessions(), fake)

	// No cookie attached.
	req := httptest.NewRequest(http.MethodPost, "/gifts", strings.NewReader(validGiftBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Zero(t, fake.calls, "the service must not run without a session")
}

func TestCreateGift_Created(t *testing.T) {
	userID := uuid.New()
	giftID := uuid.New()
	fake := &fakeGiftService{out: domain.Gift{ID: giftID, ViewToken: "tok-123"}}

	rec := postGift(t, fake, giftSessions(), userID, validGiftBody)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, 1, fake.calls)
	require.Equal(t, userID, fake.gotInput.CreatorID, "creator comes from the session, not the body")
	require.Equal(t, "Feliz cumple", fake.gotInput.Title)
	require.True(t, fake.gotInput.SingleOpen)

	var resp createGiftResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, giftID, resp.ID)
	require.Equal(t, "tok-123", resp.ViewToken)
}

func TestCreateGift_BadJSON(t *testing.T) {
	fake := &fakeGiftService{}
	rec := postGift(t, fake, giftSessions(), uuid.New(), `{not json`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidRequest, decodeErrorCode(t, rec))
	require.Zero(t, fake.calls)
}

func TestCreateGift_MissingTitle(t *testing.T) {
	fake := &fakeGiftService{}
	body := `{"title":"  ","pixel_art":{},"reveal_type":"box"}`
	rec := postGift(t, fake, giftSessions(), uuid.New(), body)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidGift, decodeErrorCode(t, rec))
	require.Zero(t, fake.calls)
}

func TestCreateGift_InvalidRevealType(t *testing.T) {
	fake := &fakeGiftService{}
	body := `{"title":"x","pixel_art":{},"reveal_type":"teleport"}`
	rec := postGift(t, fake, giftSessions(), uuid.New(), body)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidRevealType, decodeErrorCode(t, rec))
	require.Zero(t, fake.calls)
}

func TestCreateGift_EmptyPixelArt(t *testing.T) {
	fake := &fakeGiftService{}
	// pixel_art omitted entirely → empty RawMessage → invalid_pixel_art.
	body := `{"title":"x","reveal_type":"box"}`
	rec := postGift(t, fake, giftSessions(), uuid.New(), body)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidPixelArt, decodeErrorCode(t, rec))
	require.Zero(t, fake.calls)
}

func TestCreateGift_InvalidRecipientEmail(t *testing.T) {
	fake := &fakeGiftService{}
	body := `{"title":"x","pixel_art":{},"reveal_type":"box","recipient_email":"nope"}`
	rec := postGift(t, fake, giftSessions(), uuid.New(), body)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidEmail, decodeErrorCode(t, rec))
	require.Zero(t, fake.calls)
}

func TestCreateGift_ServiceError(t *testing.T) {
	fake := &fakeGiftService{err: errors.New("db down")}
	rec := postGift(t, fake, giftSessions(), uuid.New(), validGiftBody)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, codeInternalError, decodeErrorCode(t, rec))
}
