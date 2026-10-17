import { useState } from "react";
import type { FormEvent } from "react";
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
import { getComposerFieldErrors } from "./lib/formValidation";
import { isBusy } from "./lib/timeline";
import type { ComposerFieldErrors } from "./lib/formValidation";
import type { AuthMode, Tag, TimelineEntry } from "./types";
import { useCurrentTime } from "./hooks/useCurrentTime";
import { useMediaQuery } from "./hooks/useMediaQuery";
import { useSidebarState } from "./hooks/useSidebarState";
import { useComposer } from "./hooks/useComposer";
import { useTagVisibility } from "./hooks/useTagVisibility";
import { useWorkspace } from "./hooks/useWorkspace";
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
  getDayElapsedOverlay,
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
import { AuthScreen } from "./components/AuthScreen";
import { ComposerDrawer } from "./components/ComposerDrawer";
import { QuickAddBar } from "./components/QuickAddBar";
import { parseEvent } from "./features/quickAdd/parseEvent";
import { resolveDraft } from "./features/quickAdd/resolveDraft";
import { getRequestErrorMessage } from "./lib/requestErrors";
import { CalendarEntry } from "./components/CalendarEntry";
import { AllDayEvent } from "./components/AllDayEvent";
import { Sidebar } from "./components/Sidebar";
import { TagEditorDrawer } from "./components/TagEditorDrawer";
import { WeekCalendar } from "./components/WeekCalendar";
import { WeekHeader } from "./components/WeekHeader";
import { TopBar } from "./components/TopBar";
import { SettingsPage } from "./components/SettingsPage";
import { HouseholdSettings } from "./components/HouseholdSettings";
import { ShareEventDrawer } from "./components/ShareEventDrawer";
import { SharedEventDrawer } from "./components/SharedEventDrawer";
import {
  useEventShareMutations,
  useEventShares,
} from "./features/sharing/useEventShares";
import {
  useHousehold,
  useHouseholdMutations,
} from "./features/household/useHousehold";
import { TransitCardPill } from "./components/TransitCardPill";
import { TransitCardDrawer } from "./components/TransitCardDrawer";
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
import { RemindersDrawer } from "./components/RemindersDrawer";
import { ReminderToasts } from "./components/ReminderToasts";
import type { Reminder } from "./types";
import "./App.css";

const NO_REMINDERS: Reminder[] = [];

function App() {
  const {
    user,
    isLoading: authLoading,
    error,
    authenticate,
    logout,
    updateDisplayName,
  } = useAuth();
  const workspaceQuery = useWorkspace(Boolean(user));
  const entries = workspaceQuery.data?.entries ?? [];
  const tags = workspaceQuery.data?.tags ?? [];
  const loading = authLoading || workspaceQuery.isLoading;
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
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
  const [actionError, setActionError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<ComposerFieldErrors>({});
  const tagMutations = useTagMutations(setActionError, removeHiddenTag);
  const { toggleTask, deleteItem, saveItem } =
    useTimelineMutations(setActionError);
  const dates = getCalendarDates(rangeStart, dayCount);
  const transitCardQuery = useTransitCardDays(
    dateKey(dates[0]),
    dateKey(dates[dates.length - 1]),
    Boolean(user),
  );
  const saveTransitCardDays = useSaveTransitCardDays(setActionError);
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
      setActionError("");
      setFieldErrors({});
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

  const submitAuth = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void authenticate(authMode, email, password);
  };

  const handleDeleteItem = async (entry: TimelineEntry) => {
    await deleteItem(
      entry,
      () => window.confirm(`Delete “${entry.title}”?`),
      () => {
        closeComposer();
        setDraftEventStart(null);
      },
    );
  };

  const deleteTag = async (tag: Tag) => {
    await tagMutations.deleteTag(tag, () =>
      window.confirm(`Delete tag “${tag.name}”?`),
    );
  };

  const createTag = async () => {
    const name = window.prompt("Tag name")?.trim();
    if (!name) return;
    await tagMutations.createTag(name);
  };

  const createItem = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const errors = getComposerFieldErrors(
      event.currentTarget,
      composerType,
      composerTagIds,
      composerMainTagId,
    );
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      setActionError("Please correct the highlighted fields.");
      return;
    }
    setFieldErrors({});
    setActionError("");
    const form = new FormData(event.currentTarget);
    await saveItem({
      type: composerType,
      editingItem,
      form,
      mainTagId: composerMainTagId,
      selectedTagIds: composerTagIds,
      tags,
      onSaved: () => {
        closeComposer();
        setActionError("");
      },
    });
  };

  if (loading)
    return <div className="loading-screen">Loading your calendar...</div>;
  if (!user) {
    return (
      <AuthScreen
        mode={authMode}
        email={email}
        password={password}
        error={error}
        onSubmit={submitAuth}
        onEmailChange={setEmail}
        onPasswordChange={setPassword}
        onToggleMode={() =>
          setAuthMode(authMode === "login" ? "register" : "login")
        }
      />
    );
  }

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
    setActionError("");
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
        onToggleTask={toggleTask}
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
        onLogout={logout}
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
                <button className="text-button" onClick={logout}>
                  Sign out
                </button>
              </div>
            }
            openTaskCount={openTasks}
            tags={tags}
            hiddenTagIds={hiddenTagIds}
            onToggleTagVisibility={toggleTagVisibility}
            onAddTag={createTag}
            onEditTags={() => {
              setActionError("");
              setShowTagEditor(true);
            }}
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
              {actionError &&
                !showComposer &&
                !showTagEditor &&
                !showTransitCard && (
                  <p className="error action-error">{actionError}</p>
                )}
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
                    elapsed={getDayElapsedOverlay(date, today, (value) =>
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
          fieldErrors={fieldErrors}
          error={actionError}
          onSubmit={createItem}
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
          error={actionError}
          onClose={() => setShowTransitCard(false)}
          onSave={async (submission) => {
            if (await saveTransitCardDays(submission))
              setShowTransitCard(false);
          }}
        />
      )}
      {showTagEditor && (
        <TagEditorDrawer
          tags={tags}
          error={actionError}
          onClose={() => setShowTagEditor(false)}
          onUpdate={tagMutations.updateTag}
          onUpdateDefault={(tag) => tagMutations.updateDefaultTag(tag.id)}
          onDelete={deleteTag}
        />
      )}
    </main>
  );
}

export default App;
