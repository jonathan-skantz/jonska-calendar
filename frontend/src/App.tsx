import { useState } from "react";
import {
  DEFAULT_SCROLL_HOUR,
  PHONE_DAY_COUNT,
  WEEK_DAY_COUNT,
  dateKey,
  formatDate,
  formatShortDate,
  formatTime,
} from "./lib/calendar";
import { PHONE_MEDIA_QUERY } from "./lib/layout";
import { isBusy } from "./lib/timeline";
import type { Tag, TimelineEntry } from "./types";
import { useCurrentTime } from "./hooks/useCurrentTime";
import { useMediaQuery } from "./hooks/useMediaQuery";
import { useSidebarState } from "./hooks/useSidebarState";
import { useComposer } from "./features/composer/useComposer";
import { useTagVisibility } from "./features/tags/useTagVisibility";
import { useWorkspace } from "./features/timeline/useWorkspace";
import { useAuth } from "./features/auth/useAuth";
import { useTimelineMutations } from "./features/timeline/useTimelineMutations";
import { useTagMutations } from "./features/tags/useTagMutations";
import { useCalendarState } from "./features/calendar/useCalendarState";
import { useCalendarPreferences } from "./features/calendar/useCalendarPreferences";
import { getEffectiveWeekStart } from "./features/calendar/calendarPreferences";
import {
  getCalendarEntryLayout,
  getCalendarDates,
  getClickedEventStart,
  getElapsedOverlayState,
  getEntriesForDate,
  entriesOverlap,
  getCommuteSegments,
  getGatheringSegment,
  eventBodyOverlapsDate,
  getEventAdjacency,
  getEventOverlapDepth,
  getEventPreview,
  getAllDayEventSpans,
  getVisibleWeekEntries,
} from "./features/calendar/timelineView";
import type {
  CommuteSegment,
  GatheringSegment,
} from "./features/calendar/timelineView";
import { CalendarDayIndicators } from "./features/calendar/CalendarDayIndicators";
import { AuthScreen } from "./features/auth/AuthScreen";
import { ComposerDrawer } from "./features/composer/ComposerDrawer";
import { QuickAddBar } from "./features/quickAdd/QuickAddBar";
import { parseEvent } from "./features/quickAdd/parseEvent";
import { resolveDraft } from "./features/quickAdd/resolveDraft";
import { getErrorText, getRequestErrorMessage } from "./lib/requestErrors";
import { CalendarEntry } from "./features/calendar/CalendarEntry";
import { AllDayEvent } from "./features/calendar/AllDayEvent";
import { Sidebar } from "./components/Sidebar";
import { TagEditorDrawer } from "./features/tags/TagEditorDrawer";
import { WeekCalendar } from "./features/calendar/WeekCalendar";
import { WeekHeader } from "./features/calendar/WeekHeader";
import { TopBar } from "./components/TopBar";
import { SettingsPage } from "./features/settings/SettingsPage";
import { HouseholdSettings } from "./features/household/HouseholdSettings";
import { ShareEventDrawer } from "./features/sharing/ShareEventDrawer";
import { SharedEventDrawer } from "./features/sharing/SharedEventDrawer";
import {
  useEventShareMutations,
  useEventShares,
} from "./features/sharing/useEventShares";
import {
  useHousehold,
  useHouseholdMutations,
} from "./features/household/useHousehold";
import { TransitCardPill } from "./features/transitCard/TransitCardPill";
import { TransitCardDrawer } from "./features/transitCard/TransitCardDrawer";
import {
  getTransitCardDayViews,
  getTransitCardPillVariant,
} from "./features/transitCard/transitCardDays";
import {
  useSaveTransitCardDays,
  useTransitCardDays,
} from "./features/transitCard/useTransitCard";
import { useWeather } from "./features/weather/useWeather";
import { getSunIndicators } from "./features/weather/sunTimes";
import { SunIndicators } from "./features/calendar/SunIndicators";
import {
  useReminderMutations,
  useReminders,
} from "./features/reminders/useReminders";
import { useReminderNotifications } from "./features/reminders/useReminderNotifications";
import {
  getRemindersByEvent,
  isReminderUpcoming,
} from "./features/reminders/reminders";
import { RemindersDrawer } from "./features/reminders/RemindersDrawer";
import { ReminderToasts } from "./features/reminders/ReminderToasts";
import type { Reminder } from "./types";
import "./App.css";

const NO_REMINDERS: Reminder[] = [];

function App() {
  const {
    user,
    isLoading: authLoading,
    authenticate,
    logout,
    updateDisplayName,
  } = useAuth();
  const workspaceQuery = useWorkspace(Boolean(user));
  const entries = workspaceQuery.data?.entries ?? [];
  const tags = workspaceQuery.data?.tags ?? [];
  const loading = authLoading || workspaceQuery.isLoading;
  const [showTagEditor, setShowTagEditor] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [showTransitCard, setShowTransitCard] = useState(false);
  const { preferences, updatePreferences } = useCalendarPreferences(
    user?.id ?? "",
  );
  const { hiddenTagIds, toggleTagVisibility, removeHiddenTag } =
    useTagVisibility();
  const today = useCurrentTime();
  const weekStartsOn = getEffectiveWeekStart(preferences.weekStartsOn);
  const isPhone = useMediaQuery(PHONE_MEDIA_QUERY);
  const dayCount = isPhone ? PHONE_DAY_COUNT : WEEK_DAY_COUNT;
  const sidebar = useSidebarState(isPhone);
  const { rangeStart, selectedDate, setSelectedDate, moveRange } =
    useCalendarState(today, weekStartsOn, dayCount);
  const tagMutations = useTagMutations(removeHiddenTag);
  const timeline = useTimelineMutations();
  const dates = getCalendarDates(rangeStart, dayCount);
  const transitCardQuery = useTransitCardDays(
    dateKey(dates[0]),
    dateKey(dates[dates.length - 1]),
    Boolean(user),
  );
  const saveTransitCardDays = useSaveTransitCardDays();
  const weatherByDay = useWeather(Boolean(user));
  const householdQuery = useHousehold(Boolean(user));
  const household = householdQuery.data ?? null;
  const householdMutations = useHouseholdMutations();
  const [sharingEvent, setSharingEvent] = useState<TimelineEntry | null>(null);
  const [viewingSharedEvent, setViewingSharedEvent] =
    useState<TimelineEntry | null>(null);
  const remindersQuery = useReminders(Boolean(user));
  const reminders = remindersQuery.data ?? NO_REMINDERS;
  const remindersByEvent = getRemindersByEvent(reminders);
  const reminderMutations = useReminderMutations();
  const [showReminders, setShowReminders] = useState(false);
  const [reminderError, setReminderError] = useState<string | null>(null);
  const reminderNotifications = useReminderNotifications(
    reminders,
    reminderMutations.markNotified,
    preferences.timeFormat,
  );
  const getReminderItems = (eventId: string) =>
    (remindersByEvent.get(eventId) ?? []).map((reminder) => ({
      id: reminder.id,
      offsetSeconds: reminder.offset_seconds,
    }));
  const addReminder = async (eventId: string, offsetSeconds: number) =>
    setReminderError(await reminderMutations.add(eventId, offsetSeconds));
  const removeReminder = async (reminderId: string) =>
    setReminderError(await reminderMutations.remove(reminderId));
  const eventSharesQuery = useEventShares(sharingEvent?.id ?? null);
  const shareMutations = useEventShareMutations(sharingEvent?.id ?? null);
  const composer = useComposer({
    tags,
    selectedDate,
    onSelectedDateChange: setSelectedDate,
    onResetErrors: () => {
      timeline.saveItem.reset();
      timeline.deleteItem.reset();
      setReminderError(null);
    },
  });
  const {
    showComposer,
    type: composerType,
    editingItem,
    selectedTagIds: composerTagIds,
    mainTagId: composerMainTagId,
    draftEventStart,
    draft: composerDraft,
    setType: setComposerType,
    setDraftEventStart,
    setSelectedTagIds: setComposerTagIds,
    setMainTagId: setComposerMainTagId,
    openNew: openNewComposer,
    edit: editItem,
    close: closeComposerDrawer,
  } = composer;
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
      openNewComposer(undefined, {
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
    closeComposerDrawer();
  };

  // Events shared with the user are read-only, so they open a details view.
  const openEntry = (entry: TimelineEntry) => {
    setReminderError(null);
    if (entry.is_shared) setViewingSharedEvent(entry);
    else editItem(entry);
  };
  // Reminders and their notifications refer to events by id.
  const openEventById = (eventId: string) => {
    const entry = entries.find((candidate) => candidate.id === eventId);
    if (!entry) return;
    setShowReminders(false);
    openEntry(entry);
  };

  const handleDeleteItem = (entry: TimelineEntry) => {
    if (!window.confirm(`Delete “${entry.title}”?`)) return;
    timeline.deleteItem.mutate(entry, {
      onSuccess: () => {
        closeComposer();
        setDraftEventStart(null);
      },
    });
  };

  const deleteTag = (tag: Tag) => {
    if (window.confirm(`Delete tag “${tag.name}”?`)) {
      tagMutations.deleteTag.mutate(tag);
    }
  };

  const createTag = () => {
    const name = window.prompt("Tag name")?.trim();
    if (name) tagMutations.createTag.mutate(name);
  };

  const saveComposerItem = (form: FormData) =>
    timeline.saveItem.mutate(
      {
        type: composerType,
        editingItem,
        form,
        mainTagId: composerMainTagId,
        selectedTagIds: composerTagIds,
        tags,
      },
      { onSuccess: closeComposer },
    );

  const openTagEditor = () => {
    tagMutations.updateTag.reset();
    tagMutations.updateDefaultTag.reset();
    tagMutations.deleteTag.reset();
    setShowTagEditor(true);
  };

  if (loading)
    return <div className="loading-screen">Loading your calendar...</div>;
  if (!user) {
    return (
      <AuthScreen
        error={getErrorText(authenticate.error, "Unable to sign in")}
        pending={authenticate.isPending}
        onSubmit={(credentials) => authenticate.mutate(credentials)}
      />
    );
  }

  const pageError = getErrorText(
    timeline.toggleTask.error ?? tagMutations.createTag.error,
    "Unable to update",
  );
  const openTasks = entries.filter(
    (entry) => entry.type === "task" && !entry.is_completed,
  ).length;
  const weekEntries = getVisibleWeekEntries(entries, dates, hiddenTagIds);
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
  const allDayEventSpans = getAllDayEventSpans(weekEntries, dates);
  const openTransitCard = () => {
    saveTransitCardDays.reset();
    setShowTransitCard(true);
  };
  const openNewEventAt = (
    event: Parameters<typeof getClickedEventStart>[0],
    date: Date,
  ) => openNewComposer(getClickedEventStart(event, date));
  const renderGridEntry = (
    entry: TimelineEntry,
    layout: ReturnType<typeof getCalendarEntryLayout>,
    trimTop = false,
    trimBottom = false,
    overlapDepth = 0,
    commuteSegments: CommuteSegment[] = [],
    gatheringSegment: GatheringSegment | null = null,
    showBody = true,
    overlapsGhost = false,
  ) => {
    return (
      <CalendarEntry
        key={entry.id}
        entry={entry}
        startMinutes={layout.startMinutes}
        durationMinutes={layout.durationMinutes}
        trimTop={trimTop}
        trimBottom={trimBottom}
        overlapDepth={overlapDepth}
        timeFormat={preferences.timeFormat}
        continuesFromPreviousDay={layout.continuesFromPreviousDay}
        continuesToNextDay={layout.continuesToNextDay}
        commuteSegments={commuteSegments}
        gatheringSegment={gatheringSegment}
        showBody={showBody}
        overlapsGhost={overlapsGhost}
        reminders={remindersByEvent.get(entry.id)}
        onEdit={openEntry}
        onToggleTask={(entry) => timeline.toggleTask.mutate(entry)}
      />
    );
  };
  const renderDayEntries = (date: Date) => {
    const dayEntries = getEntriesForDate(weekEntries, date).filter(
      (entry) => entry.type !== "event" || !entry.is_all_day,
    );
    // Busy and not-busy (heads-up, skipped) entries are laid out separately:
    // ghosts never push busy blocks aside. Ghosts render first so busy blocks
    // paint over them.
    const ghosts = dayEntries.filter((entry) => !isBusy(entry));
    const busy = dayEntries.filter(isBusy);
    const layoutLayer = (layer: TimelineEntry[]) =>
      layer.map((entry, index) => {
        const previous = layer[index - 1];
        const next = layer[index + 1];
        const { trimTop, trimBottom } = getEventAdjacency(
          entry,
          previous,
          next,
        );
        const layout = getCalendarEntryLayout(entry, today, date);
        const overlapDepth = getEventOverlapDepth(entry, layer.slice(0, index));
        const overlapsGhost =
          isBusy(entry) && ghosts.some((ghost) => entriesOverlap(entry, ghost));
        return {
          layout,
          rendered: renderGridEntry(
            entry,
            layout,
            trimTop,
            trimBottom,
            overlapDepth,
            getCommuteSegments(entry, date),
            getGatheringSegment(entry, date),
            eventBodyOverlapsDate(entry, date),
            overlapsGhost,
          ),
        };
      });
    // Elapsed entries are dimmed together in one wrapper per layer, so a
    // dimmed busy block still paints over every ghost.
    const groupElapsed = (
      layer: ReturnType<typeof layoutLayer>,
      key: string,
    ) => {
      const elapsed = layer.filter((item) => item.layout.fullyElapsed);
      return [
        ...(elapsed.length
          ? [
              <div className="elapsed-event-group" key={key}>
                {elapsed.map((item) => item.rendered)}
              </div>,
            ]
          : []),
        ...layer
          .filter((item) => !item.layout.fullyElapsed)
          .map((item) => item.rendered),
      ];
    };
    return [
      ...groupElapsed(layoutLayer(ghosts), "elapsed-ghosts"),
      ...groupElapsed(layoutLayer(busy), "elapsed-busy"),
    ];
  };
  const renderAllDayEntries = () =>
    allDayEventSpans.map((span) => (
      <AllDayEvent
        key={span.entry.id}
        {...span}
        onEdit={openEntry}
        reminders={remindersByEvent.get(span.entry.id)}
      />
    ));
  const emailName = user.email.split("@")[0];
  const userDisplayName = user.display_name || emailName;
  const transitCardPills = transitCardViews.map((view, index) => {
    const variant = getTransitCardPillVariant(view);
    return variant ? (
      <TransitCardPill
        key={view.key}
        name={view.day?.display_name ?? userDisplayName}
        variant={variant}
        column={index}
      />
    ) : null;
  });
  const openSettings = () => {
    setShowSettings(true);
    setShowTagEditor(false);
    setShowTransitCard(false);
    closeComposer();
    if (isPhone) sidebar.close();
  };
  const rangeIncludesToday = dates.some(
    (date) => dateKey(date) === dateKey(today),
  );
  // Keep the previous hour visible so the current time isn't at the very top.
  const scrollToHour = rangeIncludesToday
    ? Math.max(DEFAULT_SCROLL_HOUR, today.getHours() - 1)
    : DEFAULT_SCROLL_HOUR;
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
        onOpenReminders={() => setShowReminders(true)}
        onOpenSettings={openSettings}
        onLogout={() => logout.mutate()}
      />
      {showSettings ? (
        <SettingsPage
          displayName={user.display_name ?? ""}
          displayNamePlaceholder={emailName}
          onDisplayNameSave={updateDisplayName}
          preferences={preferences}
          onTimeFormatChange={(timeFormat) => updatePreferences({ timeFormat })}
          onWeekStartChange={(weekStartsOn) =>
            updatePreferences({ weekStartsOn })
          }
          onSunDisplayChange={updatePreferences}
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
                <button className="text-button" onClick={() => logout.mutate()}>
                  Sign out
                </button>
              </div>
            }
            openTaskCount={openTasks}
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
              onAddItem={() => openNewComposer()}
              quickAdd={<QuickAddBar onSubmit={quickAddEvent} />}
            >
              {pageError && <p className="error action-error">{pageError}</p>}
            </WeekHeader>
            <WeekCalendar
              dates={dates}
              timeFormat={preferences.timeFormat}
              hasAllDayEvents={allDayEventSpans.length > 0}
              weatherByDay={weatherByDay}
              transitCardPills={transitCardPills}
              onTransitCardClick={
                transitCardQuery.data ? openTransitCard : undefined
              }
              renderAllDay={renderAllDayEntries}
              onDayClick={openNewEventAt}
              scrollToHour={scrollToHour}
              onSwipe={isPhone ? moveRange : undefined}
              renderDay={(date) => (
                <>
                  <SunIndicators
                    {...getSunIndicators(
                      date,
                      weatherByDay.get(dateKey(date)),
                      preferences,
                      (value) =>
                        formatTime(value, {
                          preference: preferences.timeFormat,
                        }),
                    )}
                  />
                  {renderDayEntries(date)}
                  <CalendarDayIndicators
                    preview={
                      showComposer && !editingItem
                        ? getEventPreview(
                            draftEventStart,
                            date,
                            preferences.timeFormat,
                          )
                        : null
                    }
                    elapsed={getElapsedOverlayState(date, today, (value) =>
                      formatTime(value, { preference: preferences.timeFormat }),
                    )}
                  />
                </>
              )}
            />
          </div>
        </section>
      )}
      {showComposer && (
        <ComposerDrawer
          key={`${editingItem?.id ?? "new"}-${composerType}`}
          type={composerType}
          editingItem={editingItem}
          selectedDate={selectedDate}
          draftEventStart={draftEventStart}
          tags={tags}
          selectedTagIds={composerTagIds}
          mainTagId={composerMainTagId}
          timeFormat={preferences.timeFormat}
          error={getErrorText(
            timeline.saveItem.error ?? timeline.deleteItem.error,
            "Unable to save item",
          )}
          pending={timeline.saveItem.isPending}
          onSave={saveComposerItem}
          onClose={closeComposer}
          onTypeChange={setComposerType}
          onSelectedTagIdsChange={setComposerTagIds}
          onMainTagIdChange={setComposerMainTagId}
          onShare={
            editingItem?.type === "event"
              ? () => setSharingEvent(editingItem)
              : undefined
          }
          shareCount={
            entries.find((entry) => entry.id === editingItem?.id)?.share_count
          }
          reminders={editingItem ? getReminderItems(editingItem.id) : []}
          reminderError={reminderError}
          onAddReminder={(offset) =>
            editingItem && addReminder(editingItem.id, offset)
          }
          onRemoveReminder={(reminder) => removeReminder(reminder.id)}
          draftValues={composerDraft?.values}
          draftReminderOffsets={composerDraft?.reminderOffsets}
          draftWarnings={composerDraft?.warnings}
        />
      )}
      {showComposer && editingItem && (
        <button
          className="danger-button drawer-delete-action"
          type="button"
          onClick={() => handleDeleteItem(editingItem)}
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
      {viewingSharedEvent && (
        <SharedEventDrawer
          entry={viewingSharedEvent}
          timeFormat={preferences.timeFormat}
          onClose={() => setViewingSharedEvent(null)}
          reminders={getReminderItems(viewingSharedEvent.id)}
          reminderError={reminderError}
          onAddReminder={(offset) => addReminder(viewingSharedEvent.id, offset)}
          onRemoveReminder={(reminder) => removeReminder(reminder.id)}
        />
      )}
      {showReminders && (
        <RemindersDrawer
          reminders={reminders}
          now={today}
          timeFormat={preferences.timeFormat}
          onOpenEvent={openEventById}
          onRemove={(reminder) => removeReminder(reminder.id)}
          onClose={() => setShowReminders(false)}
        />
      )}
      <ReminderToasts
        toasts={reminderNotifications.toasts}
        timeFormat={preferences.timeFormat}
        onOpenEvent={openEventById}
        onDismiss={reminderNotifications.dismiss}
      />
      {showTransitCard && (
        <TransitCardDrawer
          views={transitCardViews}
          cardCount={household?.transit_card_count}
          error={getErrorText(
            saveTransitCardDays.error,
            "Unable to save transit card days",
          )}
          pending={saveTransitCardDays.isPending}
          onClose={() => setShowTransitCard(false)}
          onSave={(submission) =>
            saveTransitCardDays.mutate(submission, {
              onSuccess: () => setShowTransitCard(false),
            })
          }
        />
      )}
      {showTagEditor && (
        <TagEditorDrawer
          tags={tags}
          error={getErrorText(
            tagMutations.updateTag.error ??
              tagMutations.updateDefaultTag.error ??
              tagMutations.deleteTag.error,
            "Unable to update tags",
          )}
          onClose={() => setShowTagEditor(false)}
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

export default App;
