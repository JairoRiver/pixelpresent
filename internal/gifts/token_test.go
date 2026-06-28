package gifts

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// base64url alphabet, no padding: the exact shape of a view token.
var viewTokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// seqChecker reports "exists" for the first existsTimes calls, then "free". It
// records every candidate it was asked about, to assert retries use new tokens.
type seqChecker struct {
	existsTimes int
	err         error
	calls       int
	seen        []string
}

func (c *seqChecker) ViewTokenExists(_ context.Context, token string) (bool, error) {
	c.calls++
	c.seen = append(c.seen, token)
	if c.err != nil {
		return false, c.err
	}
	return c.calls <= c.existsTimes, nil
}

func TestGenerateViewToken_LengthAndAlphabet(t *testing.T) {
	checker := &seqChecker{} // never collides
	token, err := GenerateViewToken(context.Background(), checker)
	require.NoError(t, err)

	// 32 bytes base64url (no padding) → 43 chars.
	require.Len(t, token, 43)
	require.Regexp(t, viewTokenRe, token)
	require.Equal(t, 1, checker.calls, "a free token needs a single uniqueness check")
}

func TestGenerateViewToken_Unique(t *testing.T) {
	// Two consecutive generations against the same checker must differ.
	checker := &seqChecker{}
	a, err := GenerateViewToken(context.Background(), checker)
	require.NoError(t, err)
	b, err := GenerateViewToken(context.Background(), checker)
	require.NoError(t, err)
	require.NotEqual(t, a, b)
}

func TestGenerateViewToken_RetriesOnCollision(t *testing.T) {
	checker := &seqChecker{existsTimes: 2} // first two candidates are taken
	token, err := GenerateViewToken(context.Background(), checker)
	require.NoError(t, err)

	require.Equal(t, 3, checker.calls, "should retry past the two collisions")
	require.Equal(t, token, checker.seen[2], "the returned token is the third candidate")
	require.NotEqual(t, checker.seen[0], checker.seen[1], "each retry uses a fresh token")
}

func TestGenerateViewToken_Exhausted(t *testing.T) {
	checker := &seqChecker{existsTimes: maxViewTokenAttempts} // always taken
	_, err := GenerateViewToken(context.Background(), checker)

	require.ErrorIs(t, err, ErrViewTokenExhausted)
	require.Equal(t, maxViewTokenAttempts, checker.calls)
}

func TestGenerateViewToken_CheckerError(t *testing.T) {
	checkErr := errors.New("db down")
	checker := &seqChecker{err: checkErr}
	_, err := GenerateViewToken(context.Background(), checker)

	require.ErrorIs(t, err, checkErr, "a checker error is propagated, not swallowed")
	require.Equal(t, 1, checker.calls)
}
