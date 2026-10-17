import { useState } from "react";
import type { FormEvent } from "react";
import { dateKey, formatDate, formatTime } from "./lib/calendar";
import { getComposerFieldErrors } from "./lib/formValidation";
import type { ComposerFieldErrors } from "./lib/formValidation";
import type { AuthMode, Tag, TimelineEntry } from "./types";
import { useCurrentTime } from "./hooks/useCurrentTime";
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
  getCommuteSegments,
  eventBodyOverlapsDate,
  getEventAdjacency,
  getEventOverlapDepth,
  getEventPreview,
  getAllDayEventSpans,
  getVisibleWeekEntries,
} from "./features/calendar/timelineView";
import type { CommuteSegment } from "./features/calendar/timelineView";
import { CalendarDayIndicators } from "./features/calendar/CalendarDayIndicators";
import { AuthScreen } from "./components/AuthScreen";
import { ComposerDrawer } from "./components/ComposerDrawer";
import { CalendarEntry } from "./components/CalendarEntry";
import { AllDayEvent } from "./components/AllDayEvent";
import { Sidebar } from "./components/Sidebar";
import { TagEditorDrawer } from "./components/TagEditorDrawer";
import { WeekCalendar } from "./components/WeekCalendar";
import { WeekHeader } from "./components/WeekHeader";
import { TopBar } from "./components/TopBar";
import { SettingsPage } from "./components/SettingsPage";
import { HouseholdSettings } from "./components/HouseholdSettings";
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
import "./App.css";

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
  const { weekStart, selectedDate, setSelectedDate, moveWeek } =
    useCalendarState(today, weekStartsOn);
  const [actionError, setActionError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<ComposerFieldErrors>({});
  const tagMutations = useTagMutations(setActionError, removeHiddenTag);
  const { toggleTask, deleteItem, saveItem } =
    useTimelineMutations(setActionError);
  const dates = getCalendarDates(weekStart);
  const transitCardQuery = useTransitCardDays(
    dateKey(dates[0]),
    dateKey(dates[dates.length - 1]),
    Boolean(user),
  );
  const saveTransitCardDays = useSaveTransitCardDays(setActionError);
  const householdQuery = useHousehold(Boolean(user));
  const household = householdQuery.data ?? null;
  const householdMutations = useHouseholdMutations();
  const composer = useComposer({
    tags,
    selectedDate,
    onSelectedDateChange: setSelectedDate,
    onResetErrors: () => {
      setActionError("");
      setFieldErrors({});
    },
  });
  const {
    showComposer,
    type: composerType,
    editingItem,
    selectedTagIds: composerTagIds,
    mainTagId: composerMainTagId,
    draftEventStart,
    setType: setComposerType,
    setDraftEventStart,
    setSelectedTagIds: setComposerTagIds,
    setMainTagId: setComposerMainTagId,
    openNew: openNewComposer,
    edit: editItem,
    close: closeComposer,
  } = composer;

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
    showBody = true,
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
        showBody={showBody}
        onEdit={editItem}
        onToggleTask={toggleTask}
      />
    );
  };
  const renderDayEntries = (date: Date) => {
    const dayEntries = getEntriesForDate(weekEntries, date).filter(
      (entry) => entry.type !== "event" || !entry.is_all_day,
    );
    const entriesWithLayout = dayEntries.map((entry, index) => {
      const previous = dayEntries[index - 1];
      const next = dayEntries[index + 1];
      const { trimTop, trimBottom } = getEventAdjacency(entry, previous, next);
      const layout = getCalendarEntryLayout(entry, today, date);
      const overlapDepth = getEventOverlapDepth(
        entry,
        dayEntries.slice(0, index),
      );
      return {
        layout,
        rendered: renderGridEntry(
          entry,
          layout,
          trimTop,
          trimBottom,
          overlapDepth,
          getCommuteSegments(entry, date),
          eventBodyOverlapsDate(entry, date),
        ),
      };
    });
    const elapsedEntries = entriesWithLayout.filter(
      (item) => item.layout.fullyElapsed,
    );
    const firstElapsedIndex = entriesWithLayout.findIndex(
      (item) => item.layout.fullyElapsed,
    );

    return entriesWithLayout.map((item, index) => {
      if (!item.layout.fullyElapsed) return item.rendered;
      if (index !== firstElapsedIndex) return null;
      return (
        <div className="elapsed-event-group" key="elapsed-event-group">
          {elapsedEntries.map((elapsedEntry) => elapsedEntry.rendered)}
        </div>
      );
    });
  };
  const renderAllDayEntries = () =>
    allDayEventSpans.map((span) => (
      <AllDayEvent key={span.entry.id} {...span} onEdit={editItem} />
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
  return (
    <main className="app-shell">
      <TopBar
        email={user.email}
        onOpenSettings={() => {
          setShowSettings(true);
          setShowTagEditor(false);
          setShowTransitCard(false);
          closeComposer();
        }}
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
          <div className="timeline-column">
            <WeekHeader
              title={`${formatDate(dates[0])} – ${formatDate(dates[6])}`}
              itemCount={weekEntries.length}
              onPrevious={() => moveWeek(-7)}
              onNext={() => moveWeek(7)}
              onAddItem={() => openNewComposer()}
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
              transitCardPills={transitCardPills}
              onTransitCardClick={
                transitCardQuery.data ? openTransitCard : undefined
              }
              renderAllDay={renderAllDayEntries}
              onDayClick={openNewEventAt}
              renderDay={(date) => (
                <>
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
          <Sidebar
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
