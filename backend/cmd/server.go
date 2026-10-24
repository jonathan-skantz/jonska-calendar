package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	// Embeds the time zone database, so the assistant can read the browser's
	// time zone on hosts without one.
	_ "time/tzdata"

	"github.com/joho/godotenv"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/assistant"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/cleanup"
	"github.com/jonathan-skantz/calendar/backend/internal/demo"
	"github.com/jonathan-skantz/calendar/backend/internal/ical"
	"github.com/jonathan-skantz/calendar/backend/internal/middleware"
	"github.com/jonathan-skantz/calendar/backend/internal/quickadd"
	"github.com/jonathan-skantz/calendar/backend/internal/subscription"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
	"github.com/jonathan-skantz/calendar/backend/internal/weather"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	// Secrets such as ANTHROPIC_API_KEY live in the untracked .env.local,
	// because .env is committed.
	if err := godotenv.Load(".env.local"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Printf("Reading .env.local: %v", err)
	}
	connURL := os.Getenv("DATABASE_URL")
	if connURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	ctx := context.Background()

	dbPool, err := db.NewDBPool(ctx, connURL)
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	log.Println("Connected to the database")

	store := db.NewStore(dbPool)
	authConfig := auth.Config{
		// Behind HTTPS, the session cookie must never travel over plain HTTP.
		SecureCookies: os.Getenv("SESSION_COOKIE_SECURE") == "true",
		AllowedEmails: strings.Split(os.Getenv("REGISTRATION_ALLOWED_EMAILS"), ","),
	}
	authHandler := auth.New(store, authConfig)
	if !authHandler.RegistrationRestricted() {
		log.Println("REGISTRATION_ALLOWED_EMAILS is not set: anyone can register")
	}
	demoHandler := demo.New(store, authHandler)
	timelineHandler := timeline.New(store, authHandler.CurrentUserID)
	weatherHandler := weather.New(store, authHandler.CurrentUserID, weather.NewClient())
	icalHandler := ical.New(store, authHandler.CurrentUserID)
	// Subscriptions may not reach private addresses unless allowed, for a
	// calendar server on the same network as this one.
	subscriptionClient := subscription.NewClient(os.Getenv("SUBSCRIPTIONS_ALLOW_PRIVATE") == "true")
	subscriptionHandler := subscription.New(store, authHandler.CurrentUserID, subscriptionClient)
	go subscriptionHandler.Refresher.Run(ctx)
	go cleanup.Run(ctx, store)

	// Quick add and the assistant stay off (503) without a key, so the rest
	// of the app runs without one.
	var quickAddClient *quickadd.Client
	var assistantClient *assistant.Client
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		quickAddClient = quickadd.NewClient(apiKey, os.Getenv("ANTHROPIC_MODEL"))
		assistantClient = assistant.NewClient(apiKey, os.Getenv("ASSISTANT_MODEL"))
	}
	quickAddHandler := quickadd.New(store, authHandler.CurrentUserID, quickAddClient)
	assistantHandler := assistant.New(store, authHandler.CurrentUserID, assistantClient)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /api/auth/demo", demoHandler.Start)
	mux.HandleFunc("GET /api/auth/me", authHandler.CurrentUser)
	mux.HandleFunc("PATCH /api/auth/me", authHandler.UpdateCurrentUser)
	mux.HandleFunc("POST /api/events", timelineHandler.CreateEvent)
	mux.HandleFunc("POST /api/events/parse", quickAddHandler.Parse)
	mux.HandleFunc("PUT /api/events/{eventID}", timelineHandler.UpdateEvent)
	mux.HandleFunc("POST /api/tasks", timelineHandler.CreateTask)
	mux.HandleFunc("PUT /api/tasks/{taskID}", timelineHandler.UpdateTask)
	mux.HandleFunc("PATCH /api/tasks/{taskID}", timelineHandler.SetTaskCompleted)
	mux.HandleFunc("DELETE /api/timeline/{itemID}", timelineHandler.DeleteTimelineItem)
	mux.HandleFunc("GET /api/timeline", timelineHandler.ListTimeline)
	mux.HandleFunc("POST /api/tags", timelineHandler.CreateTag)
	mux.HandleFunc("GET /api/tags", timelineHandler.ListTags)
	mux.HandleFunc("PATCH /api/tags/{tagID}", timelineHandler.RenameTag)
	mux.HandleFunc("DELETE /api/tags/{tagID}", timelineHandler.DeleteTag)
	mux.HandleFunc("PUT /api/tags/default", timelineHandler.SetDefaultTag)
	mux.HandleFunc("POST /api/timeline/{itemID}/tags", timelineHandler.AddTagToItem)
	mux.HandleFunc("DELETE /api/timeline/{itemID}/tags/{tagID}", timelineHandler.RemoveTagFromItem)
	mux.HandleFunc("GET /api/events/{eventID}/shares", timelineHandler.ListEventShares)
	mux.HandleFunc("POST /api/events/{eventID}/shares", timelineHandler.ShareEvent)
	mux.HandleFunc("PUT /api/events/{eventID}/shares/{shareID}", timelineHandler.UpdateEventShare)
	mux.HandleFunc("DELETE /api/events/{eventID}/shares/{shareID}", timelineHandler.DeleteEventShare)
	mux.HandleFunc("PUT /api/events/{eventID}/customization", timelineHandler.CustomizeSharedEvent)
	mux.HandleFunc("DELETE /api/events/{eventID}/customization", timelineHandler.UncustomizeSharedEvent)
	mux.HandleFunc("PUT /api/subscribed-events/{eventID}/customization", timelineHandler.CustomizeSubscribedEvent)
	mux.HandleFunc("DELETE /api/subscribed-events/{eventID}/customization", timelineHandler.UncustomizeSubscribedEvent)
	mux.HandleFunc("GET /api/reminders", timelineHandler.ListReminders)
	mux.HandleFunc("POST /api/events/{eventID}/reminders", timelineHandler.CreateReminder)
	mux.HandleFunc("DELETE /api/reminders/{reminderID}", timelineHandler.DeleteReminder)
	mux.HandleFunc("POST /api/reminders/{reminderID}/notified", timelineHandler.MarkReminderNotified)
	mux.HandleFunc("GET /api/transit-card/days", timelineHandler.ListTransitCardDays)
	mux.HandleFunc("PUT /api/transit-card/days", timelineHandler.SetTransitCardDays)
	mux.HandleFunc("GET /api/household", timelineHandler.GetHousehold)
	mux.HandleFunc("POST /api/household", timelineHandler.CreateHousehold)
	mux.HandleFunc("PATCH /api/household", timelineHandler.UpdateHousehold)
	mux.HandleFunc("POST /api/household/members", timelineHandler.AddHouseholdMember)
	mux.HandleFunc("DELETE /api/household/members/me", timelineHandler.LeaveHousehold)
	mux.HandleFunc("POST /api/import/ics", icalHandler.Import)
	mux.HandleFunc("GET /api/export.ics", icalHandler.Export)
	mux.HandleFunc("GET /api/feeds", icalHandler.ListFeeds)
	mux.HandleFunc("POST /api/feeds", icalHandler.CreateFeed)
	mux.HandleFunc("PATCH /api/feeds/{feedID}", icalHandler.UpdateFeed)
	mux.HandleFunc("POST /api/feeds/{feedID}/token", icalHandler.ResetFeedToken)
	mux.HandleFunc("DELETE /api/feeds/{feedID}", icalHandler.DeleteFeed)
	// Calendar apps fetch feeds without a session; the token is the secret.
	mux.HandleFunc("GET /api/feeds/{token}/calendar.ics", icalHandler.ServeFeed)
	mux.HandleFunc("GET /api/subscriptions", subscriptionHandler.List)
	mux.HandleFunc("POST /api/subscriptions", subscriptionHandler.Create)
	mux.HandleFunc("PATCH /api/subscriptions/{subscriptionID}", subscriptionHandler.Update)
	mux.HandleFunc("POST /api/subscriptions/{subscriptionID}/refresh", subscriptionHandler.Refresh)
	mux.HandleFunc("DELETE /api/subscriptions/{subscriptionID}", subscriptionHandler.Delete)
	mux.HandleFunc("GET /api/weather", weatherHandler.Forecast)
	mux.HandleFunc("POST /api/assistant/messages", assistantHandler.Message)
	mux.HandleFunc("POST /api/assistant/conversations/{conversationID}/actions/{actionID}", assistantHandler.Decide)

	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.API(mux),
		// Bound how long a client may take to send a request and how long an
		// idle connection stays open, so slow clients cannot hold the server.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		// Longer than an assistant turn, which may take two minutes.
		WriteTimeout: 150 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		// log.Fatal skips deferred calls, so close the pool first.
		dbPool.Close()
		log.Fatal(err)
	}
}
