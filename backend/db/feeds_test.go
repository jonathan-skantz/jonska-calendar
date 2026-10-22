package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func TestFeedsAreOwnedAndFoundByToken(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID := registerUser(t, store, "owner@example.com")
	otherID := registerUser(t, store, "other@example.com")
	otherTag, err := store.CreateTag(ctx, otherID, "Theirs")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.CreateFeed(ctx, ownerID, db.FeedFields{Name: "Work", TagID: &otherTag.ID}, "token-x", "UTC"); !errors.Is(err, db.ErrTagNotFound) {
		t.Fatalf("expected another user's tag to be refused, got %v", err)
	}
	feed, err := store.CreateFeed(ctx, ownerID, db.FeedFields{Name: "All"}, "token-a", "Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	found, err := store.FeedByToken(ctx, "token-a")
	if err != nil || found.ID != feed.ID || found.UserID != ownerID || found.TimeZone != "Europe/Stockholm" {
		t.Fatalf("unexpected feed %+v, %v", found, err)
	}

	if _, err := store.UpdateFeed(ctx, otherID, feed.ID, db.FeedFields{Name: "Mine now"}); !errors.Is(err, db.ErrFeedNotFound) {
		t.Fatalf("expected another user's update to be refused, got %v", err)
	}
	if _, err := store.UpdateFeed(ctx, ownerID, feed.ID, db.FeedFields{Name: "All", TagID: &otherTag.ID}); !errors.Is(err, db.ErrTagNotFound) {
		t.Fatalf("expected another user's tag to be refused, got %v", err)
	}
	if err := store.DeleteFeed(ctx, otherID, feed.ID); !errors.Is(err, db.ErrFeedNotFound) {
		t.Fatalf("expected another user's delete to be refused, got %v", err)
	}

	if _, err := store.ResetFeedToken(ctx, ownerID, feed.ID, "token-b"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FeedByToken(ctx, "token-a"); !errors.Is(err, db.ErrFeedNotFound) {
		t.Fatalf("expected the old token to stop working, got %v", err)
	}
	if feeds, err := store.ListFeeds(ctx, otherID); err != nil || len(feeds) != 0 {
		t.Fatalf("expected no feeds for another user, got %+v, %v", feeds, err)
	}
}

func TestListEventsForFeedFiltersByTagAndAlarms(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID := registerUser(t, store, "owner@example.com")
	work, err := store.CreateTag(ctx, userID, "Work")
	if err != nil {
		t.Fatal(err)
	}
	startsAt := time.Now().Add(24 * time.Hour).Truncate(time.Minute).UTC()
	event := func(title string) db.Event {
		return db.Event{Title: title, StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable}
	}
	meetingID, err := store.CreateEvent(ctx, userID, event("Meeting"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddTagToItem(ctx, userID, meetingID, work.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateEvent(ctx, userID, event("Dinner")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateReminder(ctx, userID, meetingID, 900); err != nil {
		t.Fatal(err)
	}

	workFeed, err := store.CreateFeed(ctx, userID, db.FeedFields{Name: "Work", TagID: &work.ID}, "work", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	events, err := store.ListEventsForFeed(ctx, workFeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Title != "Meeting" || len(events[0].ReminderOffsets) != 0 {
		t.Fatalf("expected only the work event without alarms, got %+v", events)
	}

	allFeed, err := store.CreateFeed(ctx, userID, db.FeedFields{Name: "All", IncludeAlarms: true}, "all", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	events, err = store.ListEventsForFeed(ctx, allFeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected both events, got %+v", events)
	}
	for _, event := range events {
		if event.Title == "Meeting" && len(event.ReminderOffsets) != 1 {
			t.Fatalf("expected the reminder as an alarm, got %+v", event)
		}
	}

	// A feed limited to a tag goes with the tag rather than widening.
	if err := store.DeleteTag(ctx, userID, work.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FeedByToken(ctx, "work"); !errors.Is(err, db.ErrFeedNotFound) {
		t.Fatalf("expected the tag's feed to be deleted, got %v", err)
	}
}
