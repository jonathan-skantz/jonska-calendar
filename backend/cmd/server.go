package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
	// Embeds the time zone database, so the assistant can read the browser's
	// time zone on hosts without one.
	_ "time/tzdata"

	"github.com/joho/godotenv"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/assistant"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/quickadd"
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
	authHandler := auth.New(store)
	timelineHandler := timeline.New(store, authHandler.CurrentUserID)
	weatherHandler := weather.New(store, authHandler.CurrentUserID, weather.NewClient())

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
	mux.HandleFunc("GET /api/weather", weatherHandler.Forecast)
	mux.HandleFunc("POST /api/assistant/messages", assistantHandler.Message)
	mux.HandleFunc("POST /api/assistant/conversations/{conversationID}/actions/{actionID}", assistantHandler.Decide)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
		// Bounds how long a client may take to send its request headers, so
		// slow connections cannot hold the server open indefinitely.
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		// log.Fatal skips deferred calls, so close the pool first.
		dbPool.Close()
		log.Fatal(err)
	}
}
