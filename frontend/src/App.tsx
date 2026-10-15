import { useState } from "react";
import type { FormEvent } from "react";
import { format } from "date-fns";
import { formatDate } from "./lib/calendar";
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
import {
  getCalendarEntryLayout,
  getCalendarDates,
  getClickedEventStart,
  getDayElapsedOverlay,
  getEntriesForDate,
  getEventAdjacency,
  getEventPreview,
  getVisibleWeekEntries,
} from "./features/calendar/timelineView";
import { CalendarDayIndicators } from "./features/calendar/CalendarDayIndicators";
import { AuthScreen } from "./components/AuthScreen";
import { ComposerDrawer } from "./components/ComposerDrawer";
import { CalendarEntry } from "./components/CalendarEntry";
import { Sidebar } from "./components/Sidebar";
import { TagEditorDrawer } from "./components/TagEditorDrawer";
import { WeekCalendar } from "./components/WeekCalendar";
import { WeekHeader } from "./components/WeekHeader";
import { TopBar } from "./components/TopBar";
import "./App.css";

function App() {
  const {
    user,
    isLoading: authLoading,
    error,
    authenticate,
    logout,
  } = useAuth();
  const workspaceQuery = useWorkspace(Boolean(user));
  const entries = workspaceQuery.data?.entries ?? [];
  const tags = workspaceQuery.data?.tags ?? [];
  const loading = authLoading || workspaceQuery.isLoading;
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showTagEditor, setShowTagEditor] = useState(false);
  const { hiddenTagIds, toggleTagVisibility, removeHiddenTag } = useTagVisibility();
  const today = useCurrentTime();
  const { weekStart, selectedDate, setSelectedDate, moveWeek } =
    useCalendarState(today);
  const [actionError, setActionError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<ComposerFieldErrors>({});
  const tagMutations = useTagMutations(setActionError, removeHiddenTag);
  const { toggleTask, deleteItem, saveItem } = useTimelineMutations(setActionError);
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
    await deleteItem(entry, () => window.confirm(`Delete “${entry.title}”?`), () => {
      closeComposer();
      setDraftEventStart(null);
    });
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

  const dates = getCalendarDates(weekStart);
  const openTasks = entries.filter(
    (entry) => entry.type === "task" && !entry.is_completed,
  ).length;
  const weekEntries = getVisibleWeekEntries(entries, dates, hiddenTagIds);
  const openNewEventAt = (
    event: Parameters<typeof getClickedEventStart>[0],
    date: Date,
  ) => openNewComposer(getClickedEventStart(event, date));
  const renderGridEntry = (
    entry: TimelineEntry,
    trimTop = false,
    trimBottom = false,
  ) => {
    const { startMinutes, durationMinutes, fullyElapsed } =
      getCalendarEntryLayout(entry, today);
    return (
      <CalendarEntry
        key={entry.id}
        entry={entry}
        startMinutes={startMinutes}
        durationMinutes={durationMinutes}
        trimTop={trimTop}
        trimBottom={trimBottom}
        fullyElapsed={fullyElapsed}
        onEdit={editItem}
        onToggleTask={toggleTask}
      />
    );
  };
  const renderDayEntries = (date: Date) => {
    const dayEntries = getEntriesForDate(weekEntries, date);
    return dayEntries.map((entry, index) => {
      const previous = dayEntries[index - 1];
      const next = dayEntries[index + 1];
      const { trimTop, trimBottom } = getEventAdjacency(
        entry,
        previous,
        next,
      );
      return renderGridEntry(entry, trimTop, trimBottom);
    });
  };
  return (
    <main className="app-shell">
      <TopBar email={user.email} onLogout={logout} />
      <section className="content-grid">
        <div className="timeline-column">
          <WeekHeader
            title={`${formatDate(dates[0])} – ${formatDate(dates[6])}`}
            itemCount={weekEntries.length}
            onPrevious={() => moveWeek(-7)}
            onNext={() => moveWeek(7)}
            onAddItem={() => openNewComposer()}
          >
            {actionError && !showComposer && !showTagEditor && (
              <p className="error action-error">{actionError}</p>
            )}
          </WeekHeader>
          <WeekCalendar
            dates={dates}
            onDayClick={openNewEventAt}
            renderDay={(date) => (
              <>
                {renderDayEntries(date)}
                <CalendarDayIndicators
                  preview={
                    showComposer && !editingItem
                      ? getEventPreview(draftEventStart, date)
                      : null
                  }
                  elapsed={getDayElapsedOverlay(date, today, (value) =>
                    format(value, "p"),
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
      {showComposer && (
        <ComposerDrawer
          type={composerType}
          editingItem={editingItem}
          selectedDate={selectedDate}
          draftEventStart={draftEventStart}
          tags={tags}
          selectedTagIds={composerTagIds}
          mainTagId={composerMainTagId}
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
