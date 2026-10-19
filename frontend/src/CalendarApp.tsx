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
import { proposalToDraft } from "./features/assistant/assistantApi";
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
import { TagEditorPanel } from "./features/tags/TagEditorPanel";
import { useTagMutations } from "./features/tags/useTagMutations";
import { useTagVisibility } from "./features/tags/useTagVisibility";
import { useTimelineMutations } from "./features/timeline/useTimelineMutations";
import { useWorkspace } from "./features/timeline/useWorkspace";
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
  | { kind: "sharedEvent"; entry: TimelineEntry };

// The calendar for a signed-in user: the week grid and the state shared
// across features. Each drawer is a feature container that loads and saves
// its own data. Rendered inside PreferencesProvider.
export function CalendarApp({ user }: { user: User }) {
  const { logout } = useAuthActions();
  const workspaceQuery = useWorkspace();
  const entries = workspaceQuery.data?.entries ?? [];
  const tags = workspaceQuery.data?.tags ?? [];
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
  const { rangeStart, selectedDate, setSelectedDate, moveRange } =
    useCalendarState(today, weekStartsOn, dayCount);
  const dates = getCalendarDates(rangeStart, dayCount);
  const { createTag } = useTagMutations(removeHiddenTag);
  const { toggleTask } = useTimelineMutations();
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
  // Events shared with the user are read-only, so they open a details view.
  const openEntry = (entry: TimelineEntry) => {
    if (entry.is_shared) setDrawer({ kind: "sharedEvent", entry });
    else composer.edit(entry);
  };
  // Reminders and their notifications refer to events by id.
  const openEventById = (eventId: string) => {
    const entry = entries.find((candidate) => candidate.id === eventId);
    if (!entry) return;
    setDrawer(null);
    openEntry(entry);
  };
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

  if (workspaceQuery.isLoading) return <LoadingScreen />;

  const pageError = getErrorText(
    toggleTask.error ?? createTag.error,
    "Unable to update",
  );
  const weekEntries = getVisibleWeekEntries(entries, dates, hiddenTagIds);
  const allDayEventSpans = getAllDayEventSpans(weekEntries, dates);
  const formatRangeDate = isPhone ? formatShortDate : formatDate;

  return (
    <main className="app-shell">
      <TopBar
        email={user.email}
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
            hiddenTagIds={hiddenTagIds}
            onToggleTagVisibility={toggleTagVisibility}
            onAddTag={promptForTag}
            onEditTags={() => setDrawer({ kind: "tags" })}
          />
          <div className="timeline-column">
            <WeekHeader
              title={`${formatRangeDate(dates[0])} – ${formatRangeDate(dates[dates.length - 1])}`}
              itemCount={weekEntries.length}
              onPrevious={() => moveRange(-1)}
              onNext={() => moveRange(1)}
              onAddItem={() => composer.openNew()}
              quickAdd={<QuickAddBar onSubmit={quickAddEvent} />}
            >
              {pageError && <p className="error action-error">{pageError}</p>}
            </WeekHeader>
            <WeekCalendar
              dates={dates}
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
                    key={span.entry.id}
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
                    entries={weekEntries}
                    date={date}
                    now={today}
                    remindersByEvent={remindersByEvent}
                    onEdit={openEntry}
                    onToggleTask={(entry) => toggleTask.mutate(entry)}
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
          selectedDate={selectedDate}
        />
      )}
      {drawer?.kind === "sharedEvent" && (
        <SharedEventPanel
          key={drawer.entry.id}
          entry={drawer.entry}
          onClose={closeDrawer}
        />
      )}
      {drawer?.kind === "assistant" && (
        <AssistantDrawer
          turns={assistant.turns}
          ended={assistant.ended}
          pendingText={assistant.pendingText}
          error={getErrorText(assistant.error, "Couldn't reach the assistant")}
          onSend={assistant.send}
          onReset={assistant.reset}
          onOpenProposal={(proposal) => {
            // The conversation is kept, so reopening the drawer continues it.
            setDrawer(null);
            composer.openNew(undefined, proposalToDraft(proposal));
          }}
          onClose={closeDrawer}
        />
      )}
      {drawer?.kind === "reminders" && (
        <RemindersPanel
          now={today}
          onOpenEvent={openEventById}
          onClose={closeDrawer}
        />
      )}
      <ReminderToasts
        toasts={reminderDelivery.toasts}
        onOpenEvent={openEventById}
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
