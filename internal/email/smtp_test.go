package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/stretchr/testify/require"
)

// TestSMTPSender_SendToMailpit is a manual integration test: it sends a real
// email through the SMTP server configured in config.yaml (Mailpit in dev) and
// verifies via Mailpit's HTTP API that it arrived. Run with:
//
//	MAILPIT_TEST=1 go test ./internal/email/...   (needs `task dev-up`)
func TestSMTPSender_SendToMailpit(t *testing.T) {
	if os.Getenv("MAILPIT_TEST") == "" {
		t.Skip("set MAILPIT_TEST=1 to run the manual Mailpit test (needs `task dev-up`)")
	}

	config, err := util.LoadConfig("../../config.yaml")
	require.NoError(t, err)

	sender := NewSMTPSender(SMTPConfig{
		Host:     config.Email.Host,
		Port:     config.Email.Port,
		Username: config.Email.Username,
		Password: config.Email.Password,
		From:     config.Email.From,
	})

	subject := "pixelpresent test " + util.RandomString(10)
	err = sender.Send(context.Background(), domain.Email{
		To:       util.RandomEmail(),
		Subject:  subject,
		BodyText: "plain body",
		BodyHTML: "<p>html body</p>",
	})
	require.NoError(t, err)

	require.True(t, mailpitHasSubject(t, config.Email.Host, subject),
		"email with subject %q not found in Mailpit", subject)
}

func mailpitHasSubject(t *testing.T, host, subject string) bool {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("http://%s:8025/api/v1/messages", host))
	require.NoError(t, err)
	defer resp.Body.Close()

	var payload struct {
		Messages []struct {
			Subject string `json:"Subject"`
		} `json:"messages"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))

	for _, m := range payload.Messages {
		if m.Subject == subject {
			return true
		}
	}
	return false
}
