package email

import (
	"context"
	"testing"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestFake_RecordsSentEmails(t *testing.T) {
	f := NewFake()

	_, ok := f.Last()
	require.False(t, ok)

	msg := domain.Email{To: "a@b.com", Subject: "hi", BodyText: "hello", BodyHTML: "<p>hello</p>"}
	require.NoError(t, f.Send(context.Background(), msg))

	require.Len(t, f.Sent, 1)
	last, ok := f.Last()
	require.True(t, ok)
	require.Equal(t, msg, last)
}
