package notification

import (
	"testing"

	"github.com/google/uuid"
)

func TestSkipSelfMention(t *testing.T) {
	actorID := uuid.New()
	mentionedUsers := []uuid.UUID{actorID, uuid.New(), uuid.New()}

	// Simulate the self-mention skip logic from comment service.
	notified := make([]uuid.UUID, 0)
	for _, uid := range mentionedUsers {
		if uid == actorID {
			continue // skip self-mention
		}
		notified = append(notified, uid)
	}

	if len(notified) != 2 {
		t.Errorf("expected 2 notified users (skipping self), got %d", len(notified))
	}
	for _, uid := range notified {
		if uid == actorID {
			t.Error("self-mention was not skipped")
		}
	}
}

func TestDeduplication(t *testing.T) {
	// Simulate deduplication of mentioned usernames.
	mentions := []string{"alice", "bob", "alice", "charlie", "bob"}
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, m := range mentions {
		if !seen[m] {
			seen[m] = true
			unique = append(unique, m)
		}
	}

	if len(unique) != 3 {
		t.Errorf("expected 3 unique mentions, got %d", len(unique))
	}
}
