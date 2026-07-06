package gifts

import (
	"testing"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCheckVisibility(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	ptr := func(tm time.Time) *time.Time { return &tm }

	// pub is a "published in the past" marker shared by the cases that assume the
	// gift is already published (the draft cases below leave PublishedAt nil).
	pub := ptr(past)

	tests := []struct {
		name string
		gift domain.Gift
		want Visibility
	}{
		{
			name: "unpublished (no published_at) is a draft",
			gift: domain.Gift{},
			want: NotPublished,
		},
		{
			name: "draft short-circuits every other gate",
			gift: domain.Gift{ScheduledOpenAt: ptr(future), ExpiresAt: ptr(past), SingleOpen: true, OpenedAt: ptr(past)},
			want: NotPublished,
		},
		{
			name: "published with no other gates is visible",
			gift: domain.Gift{PublishedAt: pub},
			want: Visible,
		},
		{
			name: "scheduled in the future is not yet open",
			gift: domain.Gift{PublishedAt: pub, ScheduledOpenAt: ptr(future)},
			want: NotYetOpen,
		},
		{
			name: "scheduled in the past is visible",
			gift: domain.Gift{PublishedAt: pub, ScheduledOpenAt: ptr(past)},
			want: Visible,
		},
		{
			name: "scheduled exactly now is visible (inclusive)",
			gift: domain.Gift{PublishedAt: pub, ScheduledOpenAt: ptr(now)},
			want: Visible,
		},
		{
			name: "expires in the future is visible",
			gift: domain.Gift{PublishedAt: pub, ExpiresAt: ptr(future)},
			want: Visible,
		},
		{
			name: "expires in the past is expired",
			gift: domain.Gift{PublishedAt: pub, ExpiresAt: ptr(past)},
			want: Expired,
		},
		{
			name: "expires exactly now is expired (exclusive)",
			gift: domain.Gift{PublishedAt: pub, ExpiresAt: ptr(now)},
			want: Expired,
		},
		{
			name: "single open and already opened is consumed",
			gift: domain.Gift{PublishedAt: pub, SingleOpen: true, OpenedAt: ptr(past)},
			want: AlreadyOpened,
		},
		{
			name: "single open but never opened is visible",
			gift: domain.Gift{PublishedAt: pub, SingleOpen: true},
			want: Visible,
		},
		{
			name: "opened but not single open is visible",
			gift: domain.Gift{PublishedAt: pub, SingleOpen: false, OpenedAt: ptr(past)},
			want: Visible,
		},
		{
			name: "all gates present but satisfied is visible",
			gift: domain.Gift{PublishedAt: pub, ScheduledOpenAt: ptr(past), ExpiresAt: ptr(future), SingleOpen: true},
			want: Visible,
		},
		{
			name: "not-yet-open takes precedence over expired",
			gift: domain.Gift{PublishedAt: pub, ScheduledOpenAt: ptr(future), ExpiresAt: ptr(past)},
			want: NotYetOpen,
		},
		{
			name: "expired takes precedence over already-opened",
			gift: domain.Gift{PublishedAt: pub, ExpiresAt: ptr(past), SingleOpen: true, OpenedAt: ptr(past)},
			want: Expired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, CheckVisibility(tc.gift, now))
		})
	}
}

func TestVisibilityString(t *testing.T) {
	require.Equal(t, "visible", Visible.String())
	require.Equal(t, "not_published", NotPublished.String())
	require.Equal(t, "not_yet_open", NotYetOpen.String())
	require.Equal(t, "expired", Expired.String())
	require.Equal(t, "already_opened", AlreadyOpened.String())
}
