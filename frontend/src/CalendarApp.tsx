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
import { LoadingScreen } from "./components/LoadingScreen";
import type { Tag, TimelineEntry, User } from "./types";
import { useCurrentTime } from "./hooks/useCurrentTime";
import { useMediaQuery } from "./hooks/useMediaQuery";
import { useSidebarState } from "./hooks/useSidebarState";
import { Sidebar } from "./components/Sidebar";
import { TopBar } from "./components/TopBar";
import { useAuthActions } from "./features/auth/useAuth";
import { AllDayEvent } from "./features/calendar/AllDayEvent";
import { CalendarDayIndicators } from "./features/calendar/CalendarDayIndicators";
import { DayEntries } from "./features/calendar/DayEntries";
import { SunIndicators } from "./features/calendar/SunIndicators";
import { WeekCalendar } from "./features/calendar/WeekCalendar";
import { WeekHeader } from "./features/calendar/WeekHeader";
import { getEffectiveWeekStart } from "./features/calendar/calendarPreferences";
import {
  getAllDayEventSpans,
  getCalendarDates,
  getClickedEventStart,
  getElapsedOverlayState,
  getEventPreview,
  getScrollToHour,
  getVisibleWeekEntries,
} from "./features/calendar/timelineView";
import { usePreferences } from "./features/calendar/preferencesContext";
import { useCalendarState } from "./features/calendar/useCalendarState";
import { ComposerDrawer } from "./features/composer/ComposerDrawer";
import { useComposer } from "./features/composer/useComposer";
import { HouseholdSettings } from "./features/household/HouseholdSettings";
import {
  useHousehold,
  useHouseholdMutations,
} from "./features/household/useHousehold";
import { QuickAddBar } from "./features/quickAdd/QuickAddBar";
import { parseEvent } from "./features/quickAdd/parseEvent";
import { resolveDraft } from "./features/quickAdd/resolveDraft";
import { ReminderToasts } from "./features/reminders/ReminderToasts";
import { RemindersDrawer } from "./features/reminders/RemindersDrawer";
import { isReminderUpcoming } from "./features/reminders/reminders";
import { useReminderActions } from "./features/reminders/useReminderActions";
import { useReminderDelivery } from "./features/reminders/useReminderNotifications";
import { SettingsPage } from "./features/settings/SettingsPage";
import { ShareEventDrawer } from "./features/sharing/ShareEventDrawer";
import { SharedEventDrawer } from "./features/sharing/SharedEventDrawer";
import {
  useEventShareMutations,
  useEventShares,
} from "./features/sharing/useEventShares";
import { TagEditorDrawer } from "./features/tags/TagEditorDrawer";
import { useTagMutations } from "./features/tags/useTagMutations";
import { useTagVisibility } from "./features/tags/useTagVisibility";
import { useTimelineMutations } from "./features/timeline/useTimelineMutations";
import { useWorkspace } from "./features/timeline/useWorkspace";
import { TransitCardDrawer } from "./features/transitCard/TransitCardDrawer";
import { TransitCardDayPills } from "./features/transitCard/TransitCardPill";
import { getTransitCardDayViews } from "./features/transitCard/transitCardDays";
import {
  useSaveTransitCardDays,
  useTransitCardDays,
} from "./features/transitCard/useTransitCard";
import { getSunIndicators } from "./features/weather/sunTimes";
import { useWeather } from "./features/weather/useWeather";

// The drawers opened from the calendar. Only one is open at a time; the
// composer and the share drawer stacked on it are tracked separately.
type OpenDrawer =
  | { kind: "tags" }
  | { kind: "transitCard" }
  | { kind: "reminders" }
  | { kind: "sharedEvent"; entry: TimelineEntry };

// The calendar for a signed-in user. Rendered inside PreferencesProvider, so
// it and everything below read preferences from context.
export function CalendarApp({ user }: { user: User }) {
  const { logout, updateDisplayName } = useAuthActions();
  const workspaceQuery = useWorkspace();
  const entries = workspaceQuery.data?.entries ?? [];
  const tags = workspaceQuery.data?.tags ?? [];
  const [showSettings, setShowSettings] = useState(false);
  const [drawer, setDrawer] = useState<OpenDrawer | null>(null);
  const [sharingEvent, setSharingEvent] = useState<TimelineEntry | null>(null);
  const { preferences } = usePreferences();
  const { timeFormat } = preferences;
  const formatTimeLabel = (value: Date) =>
    formatTime(value, { preference: timeFormat });
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
  const tagMutations = useTagMutations(removeHiddenTag);
  const timeline = useTimelineMutations();
  const transitCardQuery = useTransitCardDays(
    dateKey(dates[0]),
    dateKey(dates[dates.length - 1]),
  );
  const saveTransitCardDays = useSaveTransitCardDays();
  const weatherByDay = useWeather();
  const household = useHousehold().data ?? null;
  const householdMutations = useHouseholdMutations();
  const reminders = useReminderActions();
  const reminderDelivery = useReminderDelivery();
  const eventSharesQuery = useEventShares(sharingEvent?.id ?? null);
  const shareMutations = useEventShareMutations(sharingEvent?.id ?? null);
  const composer = useComposer({
    tags,
    selectedDate,
    onSelectedDateChange: setSelectedDate,
    onResetErrors: () => {
      timeline.saveItem.reset();
      timeline.deleteItem.reset();
      reminders.clearError();
    },
  });
  const { editingItem } = composer;

  // Quick add only prefills the event form; the user still reviews and saves.
  const quickAddEvent = async (text: string) => {
    try {
      const result = await parseEvent(text);
      const resolved = resolveDraft(
        result.draft,
        new Date(),
        weekStartsOn,
        selectedDate,
      );
      composer.openNew(undefined, {
        ...resolved,
        warnings: [
          ...result.unresolved.map((part) => `Couldn't place "${part}"`),
          ...result.warnings,
          ...resolved.warnings,
        ],
      });
      return null;
    } catch (error) {
      return getRequestErrorMessage(error, "Couldn't read that");
    }
  };
  // The share drawer is stacked on the composer, so it closes with it.
  const closeComposer = () => {
    setSharingEvent(null);
    composer.close();
  };
  // Events shared with the user are read-only, so they open a details view.
  const openEntry = (entry: TimelineEntry) => {
    reminders.clearError();
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
  const deleteItem = (entry: TimelineEntry) => {
    if (!window.confirm(`Delete “${entry.title}”?`)) return;
    timeline.deleteItem.mutate(entry, {
      onSuccess: () => {
        closeComposer();
        composer.setDraftEventStart(null);
      },
    });
  };
  const saveComposerItem = (form: FormData) =>
    timeline.saveItem.mutate(
      {
        type: composer.type,
        editingItem,
        form,
        mainTagId: composer.mainTagId,
        selectedTagIds: composer.selectedTagIds,
        tags,
      },
      { onSuccess: closeComposer },
    );
  const createTag = () => {
    const name = window.prompt("Tag name")?.trim();
    if (name) tagMutations.createTag.mutate(name);
  };
  const deleteTag = (tag: Tag) => {
    if (window.confirm(`Delete tag “${tag.name}”?`)) {
      tagMutations.deleteTag.mutate(tag);
    }
  };
  const openTagEditor = () => {
    tagMutations.updateTag.reset();
    tagMutations.updateDefaultTag.reset();
    tagMutations.deleteTag.reset();
    setDrawer({ kind: "tags" });
  };
  const openTransitCard = () => {
    saveTransitCardDays.reset();
    setDrawer({ kind: "transitCard" });
  };
  const openSettings = () => {
    setShowSettings(true);
    setDrawer(null);
    closeComposer();
    if (isPhone) sidebar.close();
  };
  const signOut = () => logout.mutate();

  if (workspaceQuery.isLoading) return <LoadingScreen />;

  const pageError = getErrorText(
    timeline.toggleTask.error ?? tagMutations.createTag.error,
    "Unable to update",
  );
  const weekEntries = getVisibleWeekEntries(entries, dates, hiddenTagIds);
  const allDayEventSpans = getAllDayEventSpans(weekEntries, dates);
  // Until the days load, every day would look unreported, so show nothing.
  const transitCardViews = transitCardQuery.data
    ? getTransitCardDayViews(
        dates,
        transitCardQuery.data,
        today,
        user.id,
        household?.members,
      )
    : [];
  const emailName = user.email.split("@")[0];
  const formatRangeDate = isPhone ? formatShortDate : formatDate;

  return (
    <main className="app-shell">
      <TopBar
        email={user.email}
        sidebarOpen={sidebar.open}
        onToggleSidebar={sidebar.toggle}
        upcomingReminderCount={
          reminders.reminders.filter((reminder) =>
            isReminderUpcoming(reminder, today),
          ).length
        }
        onOpenReminders={() => setDrawer({ kind: "reminders" })}
        onOpenSettings={openSettings}
        onLogout={signOut}
      />
      {showSettings ? (
        <SettingsPage
          displayName={user.display_name ?? ""}
          displayNamePlaceholder={emailName}
          onDisplayNameSave={updateDisplayName}
          onBack={() => setShowSettings(false)}
        >
          <HouseholdSettings
            household={household}
            currentUserId={user.id}
            onCreate={householdMutations.create}
            onUpdate={householdMutations.update}
            onAddMember={householdMutations.addMember}
            onLeave={householdMutations.leave}
          />
        </SettingsPage>
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
            onAddTag={createTag}
            onEditTags={openTagEditor}
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
                  views={transitCardViews}
                  fallbackName={user.display_name || emailName}
                />
              }
              onTransitCardClick={
                transitCardQuery.data ? openTransitCard : undefined
              }
              renderAllDay={() =>
                allDayEventSpans.map((span) => (
                  <AllDayEvent
                    key={span.entry.id}
                    {...span}
                    onEdit={openEntry}
                    reminders={reminders.remindersByEvent.get(span.entry.id)}
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
                    remindersByEvent={reminders.remindersByEvent}
                    onEdit={openEntry}
                    onToggleTask={(entry) => timeline.toggleTask.mutate(entry)}
                  />
                  <CalendarDayIndicators
                    preview={
                      composer.showComposer && !editingItem
                        ? getEventPreview(
                            composer.draftEventStart,
                            date,
                            timeFormat,
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
        <ComposerDrawer
          key={`${editingItem?.id ?? "new"}-${composer.type}`}
          type={composer.type}
          editingItem={editingItem}
          selectedDate={selectedDate}
          draftEventStart={composer.draftEventStart}
          tags={tags}
          selectedTagIds={composer.selectedTagIds}
          mainTagId={composer.mainTagId}
          error={getErrorText(
            timeline.saveItem.error ?? timeline.deleteItem.error,
            "Unable to save item",
          )}
          pending={timeline.saveItem.isPending}
          onSave={saveComposerItem}
          onClose={closeComposer}
          onTypeChange={composer.setType}
          onSelectedTagIdsChange={composer.setSelectedTagIds}
          onMainTagIdChange={composer.setMainTagId}
          onShare={
            editingItem?.type === "event"
              ? () => setSharingEvent(editingItem)
              : undefined
          }
          shareCount={
            entries.find((entry) => entry.id === editingItem?.id)?.share_count
          }
          reminders={editingItem ? reminders.itemsFor(editingItem.id) : []}
          reminderError={reminders.error}
          onAddReminder={(offset) =>
            editingItem && reminders.add(editingItem.id, offset)
          }
          onRemoveReminder={(reminder) => reminders.remove(reminder.id)}
          draftValues={composer.draft?.values}
          draftReminderOffsets={composer.draft?.reminderOffsets}
          draftWarnings={composer.draft?.warnings}
        />
      )}
      {composer.showComposer && editingItem && (
        <button
          className="danger-button drawer-delete-action"
          type="button"
          onClick={() => deleteItem(editingItem)}
        >
          Delete item
        </button>
      )}
      {/* Opened from the composer; closeComposer also clears it. */}
      {sharingEvent && (
        <ShareEventDrawer
          event={sharingEvent}
          shares={eventSharesQuery.data}
          onAdd={shareMutations.add}
          onUpdate={shareMutations.update}
          onRemove={async (share) =>
            window.confirm(
              `Stop sharing “${sharingEvent.title}” with ${share.recipient_display_name}?`,
            )
              ? shareMutations.remove(share.id)
              : null
          }
          onClose={() => setSharingEvent(null)}
        />
      )}
      {drawer?.kind === "sharedEvent" && (
        <SharedEventDrawer
          entry={drawer.entry}
          onClose={() => setDrawer(null)}
          reminders={reminders.itemsFor(drawer.entry.id)}
          reminderError={reminders.error}
          onAddReminder={(offset) => reminders.add(drawer.entry.id, offset)}
          onRemoveReminder={(reminder) => reminders.remove(reminder.id)}
        />
      )}
      {drawer?.kind === "reminders" && (
        <RemindersDrawer
          reminders={reminders.reminders}
          now={today}
          onOpenEvent={openEventById}
          onRemove={(reminder) => reminders.remove(reminder.id)}
          onClose={() => setDrawer(null)}
        />
      )}
      <ReminderToasts
        toasts={reminderDelivery.toasts}
        onOpenEvent={openEventById}
        onDismiss={reminderDelivery.dismiss}
      />
      {drawer?.kind === "transitCard" && (
        <TransitCardDrawer
          views={transitCardViews}
          cardCount={household?.transit_card_count}
          error={getErrorText(
            saveTransitCardDays.error,
            "Unable to save transit card days",
          )}
          pending={saveTransitCardDays.isPending}
          onClose={() => setDrawer(null)}
          onSave={(submission) =>
            saveTransitCardDays.mutate(submission, {
              onSuccess: () => setDrawer(null),
            })
          }
        />
      )}
      {drawer?.kind === "tags" && (
        <TagEditorDrawer
          tags={tags}
          error={getErrorText(
            tagMutations.updateTag.error ??
              tagMutations.updateDefaultTag.error ??
              tagMutations.deleteTag.error,
            "Unable to update tags",
          )}
          onClose={() => setDrawer(null)}
          onUpdate={(tag, name, color) => {
            if (name) tagMutations.updateTag.mutate({ tag, name, color });
          }}
          onUpdateDefault={(tag) =>
            tagMutations.updateDefaultTag.mutate(tag.id)
          }
          onDelete={deleteTag}
        />
      )}
    </main>
  );
}
