package domain

import "context"

// Email is a message to be delivered to a single recipient. BodyText and
// BodyHTML are alternative representations of the same content; either or both
// may be set.
type Email struct {
	To       string
	Subject  string
	BodyText string
	BodyHTML string
}

// EmailSender delivers emails.
type EmailSender interface {
	Send(ctx context.Context, msg Email) error
}
