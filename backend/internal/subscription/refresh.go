package subscription

import (
	"bytes"
	"context"
	"errors"
	"log"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/ical"
)

const (
	// staleAfter is how old a subscription's last fetch may get before the
	// background refresh fetches it again.
	staleAfter = 6 * time.Hour
	// checkEvery is how often the background refresh looks for stale
	// subscriptions.
	checkEvery = 15 * time.Minute
)

// Feed is a fetched calendar, ready to store.
type Feed struct {
	// Name is the calendar's own name, if it gives one.
	Name   string
	Events []db.SubscribedEvent
}

// errFeed marks an error in the feed rather than in the server, so the
// subscriber is told about it.
type errFeed struct{ err error }

func (e errFeed) Error() string { return e.err.Error() }
func (e errFeed) Unwrap() error { return e.err }

// IsFeedError reports whether err is the feed's fault: it could not be
// fetched or is not a calendar. Other errors are the server's.
func IsFeedError(err error) bool {
	_, ok := errors.AsType[errFeed](err)
	return ok
}

type Refresher struct {
	Store  *db.Store
	Client *Client
}

// Load fetches a feed and reads its events in loc, the subscriber's time
// zone. Events the app cannot show are left out.
func (r *Refresher) Load(ctx context.Context, feedURL string, loc *time.Location) (Feed, error) {
	body, err := r.Client.Fetch(ctx, feedURL)
	if err != nil {
		return Feed{}, errFeed{err}
	}
	parsed, err := ical.Parse(bytes.NewReader(body), loc)
	if err != nil {
		return Feed{}, errFeed{err}
	}
	feed := Feed{Name: parsed.Name}
	for _, event := range parsed.Events {
		// The publisher's commute and skip decisions are their own, like an
		// owner's on a shared event.
		event.CommuteBeforeSeconds, event.CommuteAfterSeconds, event.IsSkipped = 0, 0, false
		for index := range event.Exceptions {
			event.Exceptions[index].IsSkipped = nil
		}
		validated, exceptions, err := ical.Validate(event)
		if err != nil {
			continue
		}
		subscribed := db.SubscribedEvent{Event: validated, Exceptions: exceptions}
		if event.UID != "" {
			subscribed.UID = &event.UID
		}
		feed.Events = append(feed.Events, subscribed)
	}
	return feed, nil
}

// Refresh fetches a subscription again and replaces its events. When the
// feed fails, the error is recorded and the last events stay.
func (r *Refresher) Refresh(ctx context.Context, subscription db.Subscription) (db.Subscription, error) {
	loc, err := time.LoadLocation(subscription.TimeZone)
	if err != nil {
		return subscription, err
	}
	feed, err := r.Load(ctx, subscription.URL, loc)
	if IsFeedError(err) {
		updated, markErr := r.Store.MarkSubscriptionFailed(ctx, subscription.UserID, subscription.ID, Message(err))
		if markErr != nil {
			return subscription, markErr
		}
		return updated, err
	}
	if err != nil {
		return subscription, err
	}
	return r.Store.ReplaceSubscriptionEvents(ctx, subscription.UserID, subscription.ID, feed.Events)
}

// Run refreshes stale subscriptions until ctx ends, one at a time.
func (r *Refresher) Run(ctx context.Context) {
	ticker := time.NewTicker(checkEvery)
	defer ticker.Stop()
	for {
		r.refreshStale(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Refresher) refreshStale(ctx context.Context) {
	stale, err := r.Store.StaleSubscriptions(ctx, time.Now().Add(-staleAfter))
	if err != nil {
		log.Printf("listing stale subscriptions: %v", err)
		return
	}
	for _, subscription := range stale {
		if ctx.Err() != nil {
			return
		}
		if _, err := r.Refresh(ctx, subscription); err != nil {
			log.Printf("refreshing subscription %s: %v", subscription.ID, err)
		}
	}
}
