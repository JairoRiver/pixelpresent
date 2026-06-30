package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/reactions"
)

// fakeReactionService records the input and returns canned output, to test the
// HTTP layer (status mapping) in isolation from the reaction logic.
type fakeReactionService struct {
	gotInput reactions.CreateInput
	out      domain.Reaction
	err      error

	listGiftID uuid.UUID
	listOwner  uuid.UUID
	listOut    []domain.Reaction
	listErr    error
}

func (f *fakeReactionService) Create(_ context.Context, in reactions.CreateInput) (domain.Reaction, error) {
	f.gotInput = in
	return f.out, f.err
}

func (f *fakeReactionService) ListForOwner(_ context.Context, giftID, ownerID uuid.UUID) ([]domain.Reaction, error) {
	f.listGiftID = giftID
	f.listOwner = ownerID
	return f.listOut, f.listErr
}

// postReaction sends body to POST /g/{token}/reactions (no session).
func postReaction(t *testing.T, svc ReactionService, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, giftSessions(), nil, svc)
	req := httptest.NewRequest(http.MethodPost, "/api/g/"+token+"/reactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func TestCreateReaction_OK(t *testing.T) {
	fake := &fakeReactionService{out: domain.Reaction{
		ID:    uuid.New(),
		Kind:  "emoji",
		Emoji: strPtr("🎉"),
	}}

	rec := postReaction(t, fake, "secret-token", `{"kind":"emoji","emoji":"🎉"}`)

	require.Equal(t, http.StatusCreated, rec.Code)
	// The handler forwards the token from the URL and the body fields.
	require.Equal(t, "secret-token", fake.gotInput.ViewToken)
	require.Equal(t, "emoji", fake.gotInput.Kind)
	require.Equal(t, "🎉", fake.gotInput.Emoji)

	var resp reactionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, fake.out.ID, resp.ID)
	require.Equal(t, "emoji", resp.Kind)
	require.NotNil(t, resp.Emoji)
	require.Equal(t, "🎉", *resp.Emoji)
	require.Nil(t, resp.Message, "message omitted for an emoji reaction")
}

func TestCreateReaction_BadJSON(t *testing.T) {
	rec := postReaction(t, &fakeReactionService{}, "tok", `{`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidRequest, decodeErrorCode(t, rec))
}

func TestCreateReaction_Invalid(t *testing.T) {
	fake := &fakeReactionService{err: domain.ErrReactionInvalid}

	rec := postReaction(t, fake, "tok", `{"kind":"voice"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidReaction, decodeErrorCode(t, rec))
}

func TestCreateReaction_GiftNotFound(t *testing.T) {
	fake := &fakeReactionService{err: domain.ErrGiftNotFound}

	rec := postReaction(t, fake, uuid.NewString(), `{"kind":"emoji","emoji":"🎉"}`)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, codeGiftNotFound, decodeErrorCode(t, rec))
}

func TestCreateReaction_GiftNotVisible(t *testing.T) {
	fake := &fakeReactionService{err: domain.ErrGiftNotVisible}

	rec := postReaction(t, fake, "tok", `{"kind":"emoji","emoji":"🎉"}`)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, codeGiftNotVisible, decodeErrorCode(t, rec))
}

// --- GET /gifts/{id}/reactions ---

// listReactions sends GET /gifts/{id}/reactions with a valid session for userID.
func listReactions(t *testing.T, svc ReactionService, userID uuid.UUID, id string) *httptest.ResponseRecorder {
	t.Helper()
	sessions := giftSessions()
	srv := NewServer(nil, sessions, nil, svc)

	issue := httptest.NewRecorder()
	sessions.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodGet, "/api/gifts/"+id+"/reactions", nil)
	req.AddCookie(issue.Result().Cookies()[0])

	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func TestListReactions_RequiresSession(t *testing.T) {
	fake := &fakeReactionService{}
	srv := NewServer(nil, giftSessions(), nil, fake)

	req := httptest.NewRequest(http.MethodGet, "/api/gifts/"+uuid.NewString()+"/reactions", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestListReactions_Own(t *testing.T) {
	userID := uuid.New()
	giftID := uuid.New()
	fake := &fakeReactionService{listOut: []domain.Reaction{
		{ID: uuid.New(), Kind: "emoji", Emoji: strPtr("🎉")},
		{ID: uuid.New(), Kind: "text", Message: strPtr("¡gracias!")},
	}}

	rec := listReactions(t, fake, userID, giftID.String())

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, giftID, fake.listGiftID)
	require.Equal(t, userID, fake.listOwner, "scoped to the gift and the session user")

	var resp listReactionsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Reactions, 2)
	require.Equal(t, "emoji", resp.Reactions[0].Kind)
	require.Equal(t, "¡gracias!", *resp.Reactions[1].Message)
}

func TestListReactions_Empty(t *testing.T) {
	fake := &fakeReactionService{listOut: []domain.Reaction{}}
	rec := listReactions(t, fake, uuid.New(), uuid.NewString())

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"reactions":[]}`, rec.Body.String())
}

func TestListReactions_Foreign(t *testing.T) {
	fake := &fakeReactionService{listErr: domain.ErrGiftForbidden}
	rec := listReactions(t, fake, uuid.New(), uuid.NewString())

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, codeForbidden, decodeErrorCode(t, rec))
}

func TestListReactions_NotFound(t *testing.T) {
	fake := &fakeReactionService{listErr: domain.ErrGiftNotFound}
	rec := listReactions(t, fake, uuid.New(), uuid.NewString())

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, codeGiftNotFound, decodeErrorCode(t, rec))
}

func TestListReactions_InvalidID(t *testing.T) {
	fake := &fakeReactionService{}
	rec := listReactions(t, fake, uuid.New(), "not-a-uuid")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidID, decodeErrorCode(t, rec))
}
