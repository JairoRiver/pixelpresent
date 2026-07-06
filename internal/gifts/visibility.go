package gifts

import (
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
)

// Visibility is the outcome of the visibility gate: whether a gift can be shown
// to a recipient right now and, if not, why.
type Visibility int

const (
	// Visible: the gift can be shown.
	Visible Visibility = iota
	// NotPublished: still a draft; the creator has not published it yet. To a
	// recipient it is indistinguishable from a missing gift (the public endpoint
	// maps it to 404).
	NotPublished
	// NotYetOpen: scheduled_open_at is still in the future.
	NotYetOpen
	// Expired: expires_at has passed.
	Expired
	// AlreadyOpened: a single-open gift that was already opened once.
	AlreadyOpened
)

func (v Visibility) String() string {
	switch v {
	case Visible:
		return "visible"
	case NotPublished:
		return "not_published"
	case NotYetOpen:
		return "not_yet_open"
	case Expired:
		return "expired"
	case AlreadyOpened:
		return "already_opened"
	default:
		return "unknown"
	}
}

// CheckVisibility decides whether g is viewable at now. It is pure: no I/O, no
// mutation. Boundaries: scheduled_open_at is inclusive (viewable at that instant)
// and expires_at is exclusive (no longer viewable at that instant).
//
// Precedence when more than one gate would fail (only reachable through
// misconfiguration, e.g. an expiry before the open time, which gift validation
// prevents): not-published, then not-yet-open, then expired, then already-opened.
// A draft short-circuits everything: an unpublished gift never reveals its
// scheduling to a recipient.
func CheckVisibility(g domain.Gift, now time.Time) Visibility {
	if g.PublishedAt == nil {
		return NotPublished
	}
	if g.ScheduledOpenAt != nil && now.Before(*g.ScheduledOpenAt) {
		return NotYetOpen
	}
	if g.ExpiresAt != nil && !now.Before(*g.ExpiresAt) {
		return Expired
	}
	if g.SingleOpen && g.OpenedAt != nil {
		return AlreadyOpened
	}
	return Visible
}
