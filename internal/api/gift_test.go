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

	getCalls int
	gotID    uuid.UUID
	gotOwner uuid.UUID
	getOut   domain.Gift
	getErr   error

	updateCalls int
	gotUpdate   gifts.UpdateInput
	updateOut   domain.Gift
	updateErr   error

	deleteCalls int
	deleteErr   error

	listCalls int
	listOwner uuid.UUID
	listOut   []domain.Gift
	listErr   error

	viewCalls int
	gotToken  string
	viewOut   domain.Gift
	viewErr   error
}

func (f *fakeGiftService) Create(_ context.Context, in gifts.CreateInput) (domain.Gift, error) {
	f.calls++
	f.gotInput = in
	return f.out, f.err
}

func (f *fakeGiftService) GetOwned(_ context.Context, id, ownerID uuid.UUID) (domain.Gift, error) {
	f.getCalls++
	f.gotID = id
	f.gotOwner = ownerID
	return f.getOut, f.getErr
}

func (f *fakeGiftService) UpdateOwned(_ context.Context, id, ownerID uuid.UUID, in gifts.UpdateInput) (domain.Gift, error) {
	f.updateCalls++
	f.gotID = id
	f.gotOwner = ownerID
	f.gotUpdate = in
	return f.updateOut, f.updateErr
}

func (f *fakeGiftService) DeleteOwned(_ context.Context, id, ownerID uuid.UUID) error {
	f.deleteCalls++
	f.gotID = id
	f.gotOwner = ownerID
	return f.deleteErr
}

func (f *fakeGiftService) ListByOwner(_ context.Context, ownerID uuid.UUID) ([]domain.Gift, error) {
	f.listCalls++
	f.listOwner = ownerID
	return f.listOut, f.listErr
}

func (f *fakeGiftService) GetByViewToken(_ context.Context, token string) (domain.Gift, error) {
	f.viewCalls++
	f.gotToken = token
	return f.viewOut, f.viewErr
}

const giftTestSecret = "gift-secret"

func giftSessions() *auth.SessionManager {
	return auth.NewSessionManager(giftTestSecret, false, time.Hour)
}

// postGift sends body to POST /gifts with a valid session cookie for userID.
func postGift(t *testing.T, gift GiftService, sessions *auth.SessionManager, userID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, sessions, gift, nil)

	issue := httptest.NewRecorder()
	sessions.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodPost, "/api/gifts", strings.NewReader(body))
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
	srv := NewServer(nil, giftSessions(), fake, nil)

	// No cookie attached.
	req := httptest.NewRequest(http.MethodPost, "/api/gifts", strings.NewReader(validGiftBody))
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

// --- GET /gifts/{id} ---

// getGift sends GET /gifts/{id} with a valid session cookie for userID.
func getGift(t *testing.T, gift GiftService, sessions *auth.SessionManager, userID uuid.UUID, id string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, sessions, gift, nil)

	issue := httptest.NewRecorder()
	sessions.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodGet, "/api/gifts/"+id, nil)
	req.AddCookie(issue.Result().Cookies()[0])

	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func TestGetGift_RequiresSession(t *testing.T) {
	fake := &fakeGiftService{}
	srv := NewServer(nil, giftSessions(), fake, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/gifts/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Zero(t, fake.getCalls)
}

func TestGetGift_Own(t *testing.T) {
	userID := uuid.New()
	giftID := uuid.New()
	fake := &fakeGiftService{getOut: domain.Gift{
		ID:        giftID,
		CreatorID: userID,
		Title:     "Mi regalo",
		ViewToken: "tok-xyz",
	}}

	rec := getGift(t, fake, giftSessions(), userID, giftID.String())

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, fake.getCalls)
	require.Equal(t, giftID, fake.gotID)
	require.Equal(t, userID, fake.gotOwner, "ownership is checked against the session user")

	var resp giftResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, giftID, resp.ID)
	require.Equal(t, "Mi regalo", resp.Title)
	require.Equal(t, "tok-xyz", resp.ViewToken)
}

func TestGetGift_Foreign(t *testing.T) {
	fake := &fakeGiftService{getErr: domain.ErrGiftForbidden}
	rec := getGift(t, fake, giftSessions(), uuid.New(), uuid.NewString())

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, codeForbidden, decodeErrorCode(t, rec))
}

func TestGetGift_NotFound(t *testing.T) {
	fake := &fakeGiftService{getErr: domain.ErrGiftNotFound}
	rec := getGift(t, fake, giftSessions(), uuid.New(), uuid.NewString())

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, codeGiftNotFound, decodeErrorCode(t, rec))
}

func TestGetGift_InvalidID(t *testing.T) {
	fake := &fakeGiftService{}
	rec := getGift(t, fake, giftSessions(), uuid.New(), "not-a-uuid")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidID, decodeErrorCode(t, rec))
	require.Zero(t, fake.getCalls, "the service must not run on a malformed id")
}

// --- PUT /gifts/{id} ---

// putGift sends PUT /gifts/{id} with a valid session cookie for userID.
func putGift(t *testing.T, gift GiftService, sessions *auth.SessionManager, userID uuid.UUID, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, sessions, gift, nil)

	issue := httptest.NewRecorder()
	sessions.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodPut, "/api/gifts/"+id, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(issue.Result().Cookies()[0])

	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func TestUpdateGift_RequiresSession(t *testing.T) {
	fake := &fakeGiftService{}
	srv := NewServer(nil, giftSessions(), fake, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/gifts/"+uuid.NewString(), strings.NewReader(validGiftBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Zero(t, fake.updateCalls)
}

func TestUpdateGift_Own(t *testing.T) {
	userID := uuid.New()
	giftID := uuid.New()
	fake := &fakeGiftService{updateOut: domain.Gift{
		ID:        giftID,
		CreatorID: userID,
		Title:     "Feliz cumple",
		ViewToken: "tok-keep",
	}}

	rec := putGift(t, fake, giftSessions(), userID, giftID.String(), validGiftBody)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, fake.updateCalls)
	require.Equal(t, giftID, fake.gotID)
	require.Equal(t, userID, fake.gotOwner)
	require.Equal(t, "Feliz cumple", fake.gotUpdate.Title)
	require.True(t, fake.gotUpdate.SingleOpen)

	var resp giftResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, giftID, resp.ID)
	require.Equal(t, "tok-keep", resp.ViewToken)
}

func TestUpdateGift_Foreign(t *testing.T) {
	fake := &fakeGiftService{updateErr: domain.ErrGiftForbidden}
	rec := putGift(t, fake, giftSessions(), uuid.New(), uuid.NewString(), validGiftBody)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, codeForbidden, decodeErrorCode(t, rec))
}

func TestUpdateGift_NotFound(t *testing.T) {
	fake := &fakeGiftService{updateErr: domain.ErrGiftNotFound}
	rec := putGift(t, fake, giftSessions(), uuid.New(), uuid.NewString(), validGiftBody)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, codeGiftNotFound, decodeErrorCode(t, rec))
}

func TestUpdateGift_InvalidID(t *testing.T) {
	fake := &fakeGiftService{}
	rec := putGift(t, fake, giftSessions(), uuid.New(), "not-a-uuid", validGiftBody)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidID, decodeErrorCode(t, rec))
	require.Zero(t, fake.updateCalls)
}

func TestUpdateGift_InvalidBody(t *testing.T) {
	fake := &fakeGiftService{}
	// Invalid reveal_type → 400 before the service runs.
	body := `{"title":"x","pixel_art":{},"reveal_type":"teleport"}`
	rec := putGift(t, fake, giftSessions(), uuid.New(), uuid.NewString(), body)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidRevealType, decodeErrorCode(t, rec))
	require.Zero(t, fake.updateCalls)
}

// --- DELETE /gifts/{id} ---

// deleteGift sends DELETE /gifts/{id} with a valid session cookie for userID.
func deleteGift(t *testing.T, gift GiftService, sessions *auth.SessionManager, userID uuid.UUID, id string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, sessions, gift, nil)

	issue := httptest.NewRecorder()
	sessions.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodDelete, "/api/gifts/"+id, nil)
	req.AddCookie(issue.Result().Cookies()[0])

	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func TestDeleteGift_RequiresSession(t *testing.T) {
	fake := &fakeGiftService{}
	srv := NewServer(nil, giftSessions(), fake, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/gifts/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Zero(t, fake.deleteCalls)
}

func TestDeleteGift_Own(t *testing.T) {
	userID := uuid.New()
	giftID := uuid.New()
	fake := &fakeGiftService{}

	rec := deleteGift(t, fake, giftSessions(), userID, giftID.String())

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Empty(t, rec.Body.String())
	require.Equal(t, 1, fake.deleteCalls)
	require.Equal(t, giftID, fake.gotID)
	require.Equal(t, userID, fake.gotOwner)
}

func TestDeleteGift_Foreign(t *testing.T) {
	fake := &fakeGiftService{deleteErr: domain.ErrGiftForbidden}
	rec := deleteGift(t, fake, giftSessions(), uuid.New(), uuid.NewString())

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, codeForbidden, decodeErrorCode(t, rec))
}

func TestDeleteGift_NotFound(t *testing.T) {
	fake := &fakeGiftService{deleteErr: domain.ErrGiftNotFound}
	rec := deleteGift(t, fake, giftSessions(), uuid.New(), uuid.NewString())

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, codeGiftNotFound, decodeErrorCode(t, rec))
}

func TestDeleteGift_InvalidID(t *testing.T) {
	fake := &fakeGiftService{}
	rec := deleteGift(t, fake, giftSessions(), uuid.New(), "not-a-uuid")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidID, decodeErrorCode(t, rec))
	require.Zero(t, fake.deleteCalls)
}

// --- GET /gifts ---

// listGifts sends GET /gifts with a valid session cookie for userID.
func listGifts(t *testing.T, gift GiftService, sessions *auth.SessionManager, userID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, sessions, gift, nil)

	issue := httptest.NewRecorder()
	sessions.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodGet, "/api/gifts", nil)
	req.AddCookie(issue.Result().Cookies()[0])

	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func TestListGifts_RequiresSession(t *testing.T) {
	fake := &fakeGiftService{}
	srv := NewServer(nil, giftSessions(), fake, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/gifts", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Zero(t, fake.listCalls)
}

func TestListGifts_Empty(t *testing.T) {
	fake := &fakeGiftService{listOut: []domain.Gift{}}
	rec := listGifts(t, fake, giftSessions(), uuid.New())

	require.Equal(t, http.StatusOK, rec.Code)
	// Empty must serialize as [], not null.
	require.JSONEq(t, `{"gifts":[]}`, rec.Body.String())
}

func TestListGifts_ReturnsOwnGifts(t *testing.T) {
	userID := uuid.New()
	g1 := domain.Gift{ID: uuid.New(), CreatorID: userID, Title: "Uno", ViewToken: "t1", Message: "secreto"}
	g2 := domain.Gift{ID: uuid.New(), CreatorID: userID, Title: "Dos", ViewToken: "t2"}
	fake := &fakeGiftService{listOut: []domain.Gift{g1, g2}}

	rec := listGifts(t, fake, giftSessions(), userID)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, fake.listCalls)
	require.Equal(t, userID, fake.listOwner, "lists only the session user's gifts")

	var resp listGiftsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Gifts, 2)
	require.Equal(t, "Uno", resp.Gifts[0].Title)
	require.Equal(t, "t1", resp.Gifts[0].ViewToken)

	// The summary must not leak message or reveal_config.
	require.NotContains(t, rec.Body.String(), "secreto")
	require.NotContains(t, rec.Body.String(), "reveal_config")
}

func TestListGifts_ServiceError(t *testing.T) {
	fake := &fakeGiftService{listErr: errors.New("db down")}
	rec := listGifts(t, fake, giftSessions(), uuid.New())

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, codeInternalError, decodeErrorCode(t, rec))
}
