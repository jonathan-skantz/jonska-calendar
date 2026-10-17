package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
	"github.com/jonathan-skantz/calendar/backend/internal/weather"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
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

	handler := auth.New(
		db.NewStore(dbPool),
	)
	timelineHandler := timeline.New(handler.Store, func(r *http.Request) (uuid.UUID, error) {
		user, err := handler.UserFromRequest(r)
		return user.ID, err
	})

	weatherHandler := weather.New(handler.Store, weather.CurrentUser(timelineHandler.CurrentUser), weather.NewClient())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("POST /api/auth/register", handler.Register)
	mux.HandleFunc("POST /api/auth/login", handler.Login)
	mux.HandleFunc("POST /api/auth/logout", handler.Logout)
	mux.HandleFunc("GET /api/auth/me", handler.CurrentUser)
	mux.HandleFunc("PATCH /api/auth/me", handler.UpdateCurrentUser)
	mux.HandleFunc("POST /api/events", timelineHandler.CreateEvent)
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

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		dbPool.Close()
		log.Fatal(err)
	}

}
