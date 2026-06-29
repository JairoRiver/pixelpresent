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
}

func (f *fakeReactionService) Create(_ context.Context, in reactions.CreateInput) (domain.Reaction, error) {
	f.gotInput = in
	return f.out, f.err
}

// postReaction sends body to POST /g/{token}/reactions (no session).
func postReaction(t *testing.T, svc ReactionService, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewServer(nil, giftSessions(), nil, svc)
	req := httptest.NewRequest(http.MethodPost, "/g/"+token+"/reactions", strings.NewReader(body))
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
