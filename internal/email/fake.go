package email

import (
	"context"
	"sync"

	"github.com/JairoRiver/pixelpresent/internal/domain"
)

// Fake is an in-memory domain.EmailSender that records sent messages, for use
// in unit tests instead of a real SMTP server.
type Fake struct {
	mu   sync.Mutex
	Sent []domain.Email
}

var _ domain.EmailSender = (*Fake)(nil)

func NewFake() *Fake {
	return &Fake{}
}

func (f *Fake) Send(_ context.Context, msg domain.Email) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Sent = append(f.Sent, msg)
	return nil
}

// Last returns the most recently sent email and whether one exists.
func (f *Fake) Last() (domain.Email, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.Sent) == 0 {
		return domain.Email{}, false
	}
	return f.Sent[len(f.Sent)-1], true
}
