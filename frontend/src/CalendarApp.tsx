import { useState } from "react";
import {
  PHONE_DAY_COUNT,
  WEEK_DAY_COUNT,
  dateKey,
  formatDate,
  formatShortDate,
  formatTime,
} from "./lib/calendar";
import { PHONE_MEDIA_QUERY } from "./lib/layout";
import { getErrorText, getRequestErrorMessage } from "./lib/requestErrors";
import type { TimelineEntry, User } from "./types";
import { useCurrentTime } from "./hooks/useCurrentTime";
import { useMediaQuery } from "./hooks/useMediaQuery";
import { useSidebarState } from "./hooks/useSidebarState";
import { LoadingScreen } from "./components/LoadingScreen";
import { Sidebar } from "./components/Sidebar";
import { TopBar } from "./components/TopBar";
import { AssistantDrawer } from "./features/assistant/AssistantDrawer";
import { useAssistantConversation } from "./features/assistant/useAssistantConversation";
import { useAuthActions } from "./features/auth/useAuth";
import { AllDayEvent } from "./features/calendar/AllDayEvent";
import { CalendarDayIndicators } from "./features/calendar/CalendarDayIndicators";
import { DayEntries } from "./features/calendar/DayEntries";
import { SunIndicators } from "./features/calendar/SunIndicators";
import { WeekCalendar } from "./features/calendar/WeekCalendar";
import { WeekHeader } from "./features/calendar/WeekHeader";
import { getEffectiveWeekStart } from "./features/calendar/calendarPreferences";
import { usePreferences } from "./features/calendar/preferencesContext";
import {
  getAllDayEventSpans,
  getCalendarDates,
  getClickedEventStart,
  getElapsedOverlayState,
  getEventPreview,
  getScrollToHour,
  getVisibleWeekEntries,
} from "./features/calendar/timelineView";
import { useCalendarState } from "./features/calendar/useCalendarState";
import { useEntryDrag } from "./features/calendar/useEntryDrag";
import { moveEntry } from "./features/calendar/dragMove";
import { ComposerPanel } from "./features/composer/ComposerPanel";
import { useComposer } from "./features/composer/useComposer";
import { QuickAddBar } from "./features/quickAdd/QuickAddBar";
import { readEventDraft } from "./features/quickAdd/parseEvent";
import { ReminderToasts } from "./features/reminders/ReminderToasts";
import { RemindersPanel } from "./features/reminders/RemindersPanel";
import { isReminderUpcoming } from "./features/reminders/reminders";
import { useReminderActions } from "./features/reminders/useReminderActions";
import { useReminderDelivery } from "./features/reminders/useReminderNotifications";
import { SettingsScreen } from "./features/settings/SettingsScreen";
import { SharedEventPanel } from "./features/sharing/SharedEventPanel";
import { useSubscriptions } from "./features/subscriptions/useSubscriptions";
import { TagEditorPanel } from "./features/tags/TagEditorPanel";
import { useTagMutations } from "./features/tags/useTagMutations";
import { useTagVisibility } from "./features/tags/useTagVisibility";
import { useTimelineMutations } from "./features/timeline/useTimelineMutations";
import { timelineRangeFor, useTimeline } from "./features/timeline/useTimeline";
import { useRecurrenceScope } from "./features/calendar/useRecurrenceScope";
import { useTags } from "./features/tags/useTags";
import { entryKey } from "./lib/timeline";
import { TransitCardDayPills } from "./features/transitCard/TransitCardPill";
import { TransitCardPanel } from "./features/transitCard/TransitCardPanel";
import { useTransitCardViews } from "./features/transitCard/useTransitCard";
import { getSunIndicators } from "./features/weather/sunTimes";
import { useWeather } from "./features/weather/useWeather";

// The drawers opened from the calendar. Only one is open at a time; the
// composer is tracked separately.
type OpenDrawer =
  | { kind: "assistant" }
  | { kind: "tags" }
  | { kind: "transitCard" }
  | { kind: "reminders" }
  // An event shared with the user or from a calendar they subscribe to.
  | { kind: "receivedEvent"; entry: TimelineEntry };

// An event to open, and the start of the occurrence that is meant.
type EventToOpen = { eventId: string; startsAt: string };

// The calendar for a signed-in user: the week grid and the state shared
// across features. Each drawer is a feature container that loads and saves
// its own data. Rendered inside PreferencesProvider.
export function CalendarApp({ user }: { user: User }) {
  const { logout } = useAuthActions();
  const [showSettings, setShowSettings] = useState(false);
  const [drawer, setDrawer] = useState<OpenDrawer | null>(null);
  const { preferences } = usePreferences();
  const formatTimeLabel = (value: Date) =>
    formatTime(value, { preference: preferences.timeFormat });
  const { hiddenTagIds, toggleTagVisibility, removeHiddenTag } =
    useTagVisibility();
  const today = useCurrentTime();
  const weekStartsOn = getEffectiveWeekStart(preferences.weekStartsOn);
  const isPhone = useMediaQuery(PHONE_MEDIA_QUERY);
  const dayCount = isPhone ? PHONE_DAY_COUNT : WEEK_DAY_COUNT;
  const sidebar = useSidebarState(isPhone);
  const {
    rangeStart,
    selectedDate,
    setSelectedDate,
    moveRange,
    goToToday,
    goToDate,
  } = useCalendarState(today, weekStartsOn, dayCount);
  const dates = getCalendarDates(rangeStart, dayCount);
  const timelineQuery = useTimeline(timelineRangeFor(dates));
  const tagsQuery = useTags();
  const entries = timelineQuery.data ?? [];
  const tags = tagsQuery.data ?? [];
  const subscriptions = useSubscriptions().data ?? [];
  const { createTag } = useTagMutations(removeHiddenTag);
  const { toggleTask, moveItem } = useTimelineMutations();
  const scopePrompt = useRecurrenceScope();
  // A dragged occurrence of a repeating event asks which occurrences move;
  // cancelling puts it back.
  const { drag, startDrag } = useEntryDrag(async (moved) => {
    const scope = moved.occurrence_start
      ? await scopePrompt.ask({ action: "move", allowThis: true })
      : undefined;
    if (scope === null) return;
    return moveItem.mutateAsync({ entry: moved, scope });
  });
  const transitCardViews = useTransitCardViews(dates, today, user.id);
  const weatherByDay = useWeather();
  const { reminders, remindersByEvent } = useReminderActions();
  const reminderDelivery = useReminderDelivery();
  const composer = useComposer({
    tags,
    selectedDate,
    onSelectedDateChange: setSelectedDate,
  });
  const { editingItem } = composer;
  const assistant = useAssistantConversation(weekStartsOn);

  // Quick add only prefills the event form; the user still reviews and saves.
  const quickAddEvent = async (text: string) => {
    try {
      composer.openNew(
        undefined,
        await readEventDraft(text, weekStartsOn, selectedDate),
      );
      return null;
    } catch (error) {
      return getRequestErrorMessage(error, "Couldn't read that");
    }
  };
  // Shared and subscribed events are read-only, so they open a details view.
  const openEntry = (entry: TimelineEntry) => {
    if (entry.is_shared || entry.subscription)
      setDrawer({ kind: "receivedEvent", entry });
    else composer.edit(entry);
  };
  // Reminders, their notifications and the assistant open an event by its id
  // and the start of the occurrence they mean. One outside the loaded weeks
  // opens once the calendar has moved to it and loaded it.
  const [pendingOpen, setPendingOpen] = useState<EventToOpen | null>(null);
  const findEvent = ({ eventId, startsAt }: EventToOpen) =>
    entries.find(
      (candidate) =>
        candidate.id === eventId &&
        (!candidate.occurrence_start ||
          new Date(candidate.starts_at ?? "").getTime() ===
            new Date(startsAt).getTime()),
    );
  const openEvent = (eventId: string, startsAt: string) => {
    setDrawer(null);
    const entry = findEvent({ eventId, startsAt });
    if (entry) {
      openEntry(entry);
      return;
    }
    goToDate(new Date(startsAt));
    setPendingOpen({ eventId, startsAt });
  };
  // Once the weeks of an event to open have loaded, open it, or give up
  // when it is not there.
  if (pendingOpen && timelineQuery.data && !timelineQuery.isPlaceholderData) {
    const entry = findEvent(pendingOpen);
    setPendingOpen(null);
    if (entry) openEntry(entry);
  }
  const promptForTag = () => {
    const name = window.prompt("Tag name")?.trim();
    if (name) createTag.mutate(name);
  };
  const openSettings = () => {
    setShowSettings(true);
    setDrawer(null);
    composer.close();
    if (isPhone) sidebar.close();
  };
  const closeDrawer = () => setDrawer(null);
  const signOut = () => logout.mutate();

  if (timelineQuery.isLoading || tagsQuery.isLoading) return <LoadingScreen />;

  const pageError = getErrorText(
    toggleTask.error ?? moveItem.error ?? createTag.error,
    "Unable to update",
  );
  const weekEntries = getVisibleWeekEntries(entries, dates, hiddenTagIds);
  const allDayEventSpans = getAllDayEventSpans(weekEntries, dates);
  // The dragged item is drawn where it would land, through the same layout.
  const draggedWeekEntries = drag
    ? weekEntries.map((entry) =>
        entryKey(entry) === drag.entryKey
          ? moveEntry(entry, drag.start)
          : entry,
      )
    : weekEntries;
  const formatRangeDate = isPhone ? formatShortDate : formatDate;

  return (
    <main className="app-shell">
      <TopBar
        email={user.email}
        displayName={user.display_name}
        demoExpiresAt={user.demo_expires_at}
        sidebarOpen={sidebar.open}
        onToggleSidebar={sidebar.toggle}
        upcomingReminderCount={
          reminders.filter((reminder) => isReminderUpcoming(reminder, today))
            .length
        }
        onOpenAssistant={() => setDrawer({ kind: "assistant" })}
        onOpenReminders={() => setDrawer({ kind: "reminders" })}
        onOpenSettings={openSettings}
        onLogout={signOut}
      />
      {showSettings ? (
        <SettingsScreen user={user} onBack={() => setShowSettings(false)} />
      ) : (
        <section className="content-grid">
          <Sidebar
            open={sidebar.open}
            variant={isPhone ? "drawer" : "rail"}
            onClose={sidebar.close}
            drawerFooter={
              <div className="sidebar-account">
                <span className="user-email">{user.email}</span>
                <button className="text-button" onClick={openSettings}>
                  Settings
                </button>
                <button className="text-button" onClick={signOut}>
                  Sign out
                </button>
              </div>
            }
            openTaskCount={
              entries.filter(
                (entry) => entry.type === "task" && !entry.is_completed,
              ).length
            }
            tags={tags}
            subscriptions={subscriptions}
            hiddenTagIds={hiddenTagIds}
            onToggleTagVisibility={toggleTagVisibility}
            onAddTag={promptForTag}
            onEditTags={() => setDrawer({ kind: "tags" })}
          />
          <div className="timeline-column">
            <WeekHeader
              title={`${formatRangeDate(dates[0])} – ${formatRangeDate(dates[dates.length - 1])}`}
              onPrevious={() => moveRange(-1)}
              onToday={goToToday}
              onNext={() => moveRange(1)}
              onAddItem={() => composer.openNew()}
              quickAdd={<QuickAddBar onSubmit={quickAddEvent} />}
            >
              {pageError && <p className="error action-error">{pageError}</p>}
            </WeekHeader>
            <WeekCalendar
              dates={dates}
              today={today}
              hasAllDayEvents={allDayEventSpans.length > 0}
              weatherByDay={weatherByDay}
              transitCardPills={
                <TransitCardDayPills
                  views={transitCardViews ?? []}
                  fallbackName={user.display_name || user.email.split("@")[0]}
                />
              }
              onTransitCardClick={
                transitCardViews
                  ? () => setDrawer({ kind: "transitCard" })
                  : undefined
              }
              renderAllDay={() =>
                allDayEventSpans.map((span) => (
                  <AllDayEvent
                    key={entryKey(span.entry)}
                    {...span}
                    onEdit={openEntry}
                    reminders={remindersByEvent.get(span.entry.id)}
                  />
                ))
              }
              onDayClick={(event, date) =>
                composer.openNew(getClickedEventStart(event, date))
              }
              scrollToHour={getScrollToHour(dates, today)}
              onSwipe={isPhone ? moveRange : undefined}
              renderDay={(date) => (
                <>
                  <SunIndicators
                    {...getSunIndicators(
                      date,
                      weatherByDay.get(dateKey(date)),
                      preferences,
                      formatTimeLabel,
                    )}
                  />
                  <DayEntries
                    entries={draggedWeekEntries}
                    date={date}
                    now={today}
                    remindersByEvent={remindersByEvent}
                    onEdit={openEntry}
                    onToggleTask={(entry) => toggleTask.mutate(entry)}
                    onDragStart={startDrag}
                    draggingKey={drag?.dropped ? undefined : drag?.entryKey}
                  />
                  <CalendarDayIndicators
                    preview={
                      composer.showComposer && !editingItem
                        ? getEventPreview(
                            composer.draftEventStart,
                            date,
                            preferences.timeFormat,
                          )
                        : null
                    }
                    elapsed={getElapsedOverlayState(
                      date,
                      today,
                      formatTimeLabel,
                    )}
                  />
                </>
              )}
            />
          </div>
        </section>
      )}
      {composer.showComposer && (
        <ComposerPanel
          key={`${editingItem?.id ?? "new"}-${composer.type}`}
          composer={composer}
          entries={entries}
          selectedDate={selectedDate}
        />
      )}
      {drawer?.kind === "receivedEvent" && (
        <SharedEventPanel
          key={entryKey(drawer.entry)}
          // The loaded entry, so the user's own changes show once saved.
          entry={
            entries.find(
              (entry) => entryKey(entry) === entryKey(drawer.entry),
            ) ?? drawer.entry
          }
          onClose={closeDrawer}
        />
      )}
      {drawer?.kind === "assistant" && (
        <AssistantDrawer
          conversation={assistant}
          onOpenEvent={openEvent}
          onClose={closeDrawer}
        />
      )}
      {drawer?.kind === "reminders" && (
        <RemindersPanel
          now={today}
          onOpenEvent={openEvent}
          onClose={closeDrawer}
        />
      )}
      {scopePrompt.dialog}
      <ReminderToasts
        toasts={reminderDelivery.toasts}
        onOpenEvent={openEvent}
        onDismiss={reminderDelivery.dismiss}
      />
      {drawer?.kind === "transitCard" && (
        <TransitCardPanel
          dates={dates}
          today={today}
          userId={user.id}
          onClose={closeDrawer}
        />
      )}
      {drawer?.kind === "tags" && (
        <TagEditorPanel onClose={closeDrawer} onTagDeleted={removeHiddenTag} />
      )}
    </main>
  );
}
