import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { FormEvent, MouseEvent } from "react";
import { addDays, format } from "date-fns";
import {
  CALENDAR_GRID_SCALE,
  EVENT_SEPARATOR_PIXELS,
  HOURS_PER_DAY,
  LAST_EVENT_START_HOUR,
  MINIMUM_ENTRY_DURATION_MINUTES,
  MINUTES_PER_HOUR,
  PREVIEW_EVENT_DURATION_MINUTES,
  TASK_DISPLAY_DURATION_MINUTES,
  WEEK_DAY_COUNT,
  dateKey,
  formatDate,
  formatTime,
  inputDateTime,
  localDateKey,
  minutesIntoDay,
  startOfCalendarWeek,
} from "./lib/calendar";
import { api } from "./lib/api";
import { tagTextColor } from "./lib/colors";
import { getInvalidComposerFields } from "./lib/formValidation";
import { getRequestErrorMessage } from "./lib/requestErrors";
import {
  buildEventPayload,
  buildTaskPayload,
  getBackToBackTrim,
  getEntryMinutes,
  getElapsedOverlayState,
  getMainTag,
  getOrderedStripTags,
  getTimelineEntryDate,
  isEventVisible,
} from "./lib/timeline";
import type { AuthMode, Tag, TimelineEntry, User } from "./types";
import { useCurrentTime } from "./hooks/useCurrentTime";
import { useComposer } from "./hooks/useComposer";
import { useTagVisibility } from "./hooks/useTagVisibility";
import { useWorkspace } from "./hooks/useWorkspace";
import { AuthScreen } from "./components/AuthScreen";
import { ComposerDrawer } from "./components/ComposerDrawer";
import { Sidebar } from "./components/Sidebar";
import { TagEditorDrawer } from "./components/TagEditorDrawer";
import { WeekCalendar } from "./components/WeekCalendar";
import { WeekHeader } from "./components/WeekHeader";
import { TopBar } from "./components/TopBar";
import { queryKeys } from "./lib/queryKeys";
import "./App.css";

function App() {
  const queryClient = useQueryClient();
  const currentUserQuery = useQuery({
    queryKey: queryKeys.currentUser,
    queryFn: () => api<User>("/api/auth/me"),
    retry: false,
  });
  const workspaceQuery = useWorkspace(Boolean(currentUserQuery.data));
  const user = currentUserQuery.data ?? null;
  const entries = workspaceQuery.data?.entries ?? [];
  const tags = workspaceQuery.data?.tags ?? [];
  const loading = currentUserQuery.isLoading || workspaceQuery.isLoading;
  const [error, setError] = useState("");
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showTagEditor, setShowTagEditor] = useState(false);
  const { hiddenTagIds, toggleTagVisibility, removeHiddenTag } =
    useTagVisibility();
  const today = useCurrentTime();
  const [weekStart, setWeekStart] = useState(() => startOfCalendarWeek(today));
  const [selectedDate, setSelectedDate] = useState(() => dateKey(today));
  const [actionError, setActionError] = useState("");
  const [invalidFields, setInvalidFields] = useState<string[]>([]);
  const composer = useComposer({
    tags,
    selectedDate,
    onSelectedDateChange: setSelectedDate,
    onResetErrors: () => {
      setActionError("");
      setInvalidFields([]);
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

  const submitAuth = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    try {
      const path = `/api/auth/${authMode === "login" ? "login" : "register"}`;
      await api<User>(path, {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      if (authMode === "register") {
        await api("/api/auth/login", {
          method: "POST",
          body: JSON.stringify({ email, password }),
        });
      }
      await queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
      await queryClient.invalidateQueries({ queryKey: queryKeys.workspace });
    } catch (requestError) {
      setError(getRequestErrorMessage(requestError, "Unable to sign in"));
    }
  };

  const logout = async () => {
    await api("/api/auth/logout", { method: "POST" });
    await queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
    queryClient.removeQueries({ queryKey: queryKeys.workspace });
  };

  const toggleTask = async (entry: TimelineEntry) => {
    try {
      await api(`/api/tasks/${entry.id}`, {
        method: "PATCH",
        body: JSON.stringify({ is_completed: !entry.is_completed }),
      });
      await queryClient.invalidateQueries({ queryKey: queryKeys.workspace });
    } catch (requestError) {
      setActionError(getRequestErrorMessage(requestError, "Unable to update task"));
    }
  };

  const deleteItem = async (entry: TimelineEntry) => {
    if (!window.confirm(`Delete “${entry.title}”?`)) return;
    try {
      await api(`/api/timeline/${entry.id}`, { method: "DELETE" });
      await queryClient.invalidateQueries({ queryKey: queryKeys.workspace });
      closeComposer();
      setDraftEventStart(null);
    } catch (requestError) {
      setActionError(getRequestErrorMessage(requestError, "Unable to delete item"));
    }
  };

  useEffect(() => {
    if (
      !showComposer ||
      editingItem ||
      composerType !== "event" ||
      !draftEventStart
    )
      return;
    const startsInput = document.querySelector<HTMLInputElement>(
      '.composer input[name="starts_at"]',
    );
    const endsInput = document.querySelector<HTMLInputElement>(
      '.composer input[name="ends_at"]',
    );
    const start = new Date(draftEventStart);
    if (startsInput) startsInput.value = inputDateTime(draftEventStart);
    if (endsInput)
      endsInput.value = inputDateTime(
        new Date(start.getTime() + PREVIEW_EVENT_DURATION_MINUTES * MINUTES_PER_HOUR * 60 * 1000).toISOString(),
      );
  }, [composerType, draftEventStart, editingItem, showComposer]);

  useEffect(() => {
    const composer = document.querySelector(".composer");
    const form = composer?.querySelector("form");
    if (form instanceof HTMLFormElement) form.noValidate = true;
    composer?.querySelectorAll("input, select, textarea").forEach((control) => {
      control
        .closest("label")
        ?.classList.toggle(
          "field-invalid",
          invalidFields.includes((control as HTMLInputElement).name),
        );
    });
    composer
      ?.querySelector(".tag-picker")
      ?.classList.toggle(
        "field-invalid",
        invalidFields.includes("event-tags") ||
          invalidFields.includes("main-tag"),
      );
  }, [invalidFields, showComposer]);

  const updateDefaultTag = async (tagID: string) => {
    try {
      await api("/api/tags/default", {
        method: "PUT",
        body: JSON.stringify({ tag_id: tagID }),
      });
      await queryClient.invalidateQueries({ queryKey: queryKeys.workspace });
    } catch (requestError) {
      setActionError(getRequestErrorMessage(requestError, "Unable to set default tag"));
    }
  };

  const updateTag = async (event: FormEvent<HTMLFormElement>, tag: Tag) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const name = String(form.get("name") || "").trim();
    const color = String(form.get("color") || tag.color);
    if (!name) return;
    try {
      await api(`/api/tags/${tag.id}`, {
        method: "PATCH",
        body: JSON.stringify({ name, color }),
      });
      await queryClient.invalidateQueries({ queryKey: queryKeys.workspace });
    } catch (requestError) {
      setActionError(getRequestErrorMessage(requestError, "Unable to update tag"));
    }
  };

  const deleteTag = async (tag: Tag) => {
    if (!window.confirm(`Delete tag “${tag.name}”?`)) return;
    try {
      await api(`/api/tags/${tag.id}`, { method: "DELETE" });
      removeHiddenTag(tag.id);
      await queryClient.invalidateQueries({ queryKey: queryKeys.workspace });
    } catch (requestError) {
      setActionError(getRequestErrorMessage(requestError, "Unable to delete tag"));
    }
  };

  const saveEventTags = async (
    itemID: string,
    previous: Tag[],
    next: string[],
  ) => {
    const previousIDs = new Set(previous.map((tag) => tag.id));
    const nextIDs = new Set(next);
    await Promise.all([
      ...next
        .filter((tagID) => !previousIDs.has(tagID))
        .map((tagID) =>
          api(`/api/timeline/${itemID}/tags`, {
            method: "POST",
            body: JSON.stringify({ tag_id: tagID }),
          }),
        ),
      ...previous
        .filter((tag) => !nextIDs.has(tag.id))
        .map((tag) =>
          api(`/api/timeline/${itemID}/tags/${tag.id}`, { method: "DELETE" }),
        ),
    ]);
  };

  const createTag = async () => {
    const name = window.prompt("Tag name")?.trim();
    if (!name) return;
    try {
      await api<Tag>("/api/tags", {
        method: "POST",
        body: JSON.stringify({ name }),
      });
      await queryClient.invalidateQueries({ queryKey: queryKeys.workspace });
    } catch (requestError) {
      setActionError(getRequestErrorMessage(requestError, "Unable to create tag"));
    }
  };

  const createItem = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const invalidNames = getInvalidComposerFields(
      event.currentTarget,
      composerType,
      composerTagIds,
      composerMainTagId,
    );
    if (invalidNames.length > 0) {
      setInvalidFields([...new Set(invalidNames)]);
      setActionError("Please correct the highlighted fields.");
      return;
    }
    setInvalidFields([]);
    setActionError("");
    const form = new FormData(event.currentTarget);
    const payload =
      composerType === "event"
        ? buildEventPayload(form)
        : buildTaskPayload(form);
    try {
      const path = editingItem
        ? `/api/${composerType === "event" ? "events" : "tasks"}/${editingItem.id}`
        : composerType === "event"
          ? "/api/events"
          : "/api/tasks";
      if (composerType === "event" && editingItem) {
        await saveEventTags(
          editingItem.id,
          editingItem.tags ?? [],
          composerTagIds,
        );
      }
      const response = await api<{ id: string } | undefined>(path, {
        method: editingItem ? "PUT" : "POST",
        body: JSON.stringify(
          composerType === "event"
            ? { ...payload, main_tag_id: composerMainTagId }
            : payload,
        ),
      });
      if (composerType === "event") {
        const existingTags =
          editingItem?.tags ??
          (composerMainTagId
            ? tags.filter((tag) => tag.id === composerMainTagId)
            : tags.filter((tag) => tag.is_default));
        if (!editingItem)
          await saveEventTags(response?.id ?? "", existingTags, composerTagIds);
      }
      await queryClient.invalidateQueries({ queryKey: queryKeys.workspace });
      closeComposer();
      setActionError("");
    } catch (requestError) {
      setActionError(getRequestErrorMessage(requestError, "Unable to create item"));
    }
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

  const dates = Array.from({ length: WEEK_DAY_COUNT }, (_, index) =>
    addDays(weekStart, index),
  );
  const moveWeek = (days: number) => {
    const nextWeek = addDays(weekStart, days);
    setWeekStart(nextWeek);
    setSelectedDate(dateKey(nextWeek));
  };
  const openTasks = entries.filter(
    (entry) => entry.type === "task" && !entry.is_completed,
  ).length;
  const weekEntries = entries.filter((entry) => {
    const value = getTimelineEntryDate(entry);
    const inWeek = dates.some((date) => dateKey(date) === localDateKey(value));
    return inWeek && isEventVisible(entry, hiddenTagIds);
  });
  const openNewEventAt = (event: MouseEvent<HTMLDivElement>, date: Date) => {
    const bounds = event.currentTarget.getBoundingClientRect();
    const hour = Math.max(
      0,
      Math.min(
        LAST_EVENT_START_HOUR,
        Math.round((event.clientY - bounds.top) / (bounds.height / HOURS_PER_DAY)),
      ),
    );
    const start = new Date(date);
    start.setHours(hour, 0, 0, 0);
    openNewComposer(start);
  };
  const draftStartDate = draftEventStart ? new Date(draftEventStart) : null;
  const renderPreview = (date: Date) =>
    showComposer &&
    !editingItem &&
    draftStartDate &&
    localDateKey(draftEventStart ?? undefined) === dateKey(date) ? (
      <div
        className="grid-entry event preview"
        style={{
          top: `${(minutesIntoDay(draftEventStart ?? undefined) / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`,
          height: "4rem",
        }}
        aria-hidden="true"
      >
        <strong>New event</strong>
        <span>
          {formatTime(draftEventStart ?? undefined)} –{" "}
          {formatTime(
            new Date(
              draftStartDate.getTime() + PREVIEW_EVENT_DURATION_MINUTES * MINUTES_PER_HOUR * 60 * 1000,
            ).toISOString(),
          )}
        </span>
      </div>
    ) : null;
  const renderGridEntry = (
    entry: TimelineEntry,
    trimTop = false,
    trimBottom = false,
  ) => {
    const start = getEntryMinutes(entry);
    const end =
      entry.type === "event"
        ? minutesIntoDay(entry.ends_at)
        : start + TASK_DISPLAY_DURATION_MINUTES;
    const duration = Math.max(
      entry.type === "event" && end > start
        ? end - start
        : TASK_DISPLAY_DURATION_MINUTES,
      MINIMUM_ENTRY_DURATION_MINUTES,
    );
    const fullyElapsed =
      entry.type === "event" &&
      Boolean(entry.ends_at && new Date(entry.ends_at).getTime() <= today.getTime());
    const mainTag = getMainTag(entry);
    const orderedStripTags = getOrderedStripTags(entry);
    const stripTags = orderedStripTags.filter((tag) => tag.id !== mainTag?.id);
    const stripColors = orderedStripTags.map((tag) => tag.color);
    const additionalStripWidth = stripTags.length ? 50 / stripTags.length : 0;
    const stripStops = stripTags.length
      ? [
          `${mainTag?.color} 0%, ${mainTag?.color} 50%`,
          ...stripTags.map(
            (tag, index) =>
              `${tag.color} ${50 + index * additionalStripWidth}%, ${tag.color} ${50 + (index + 1) * additionalStripWidth}%`,
          ),
        ]
      : stripColors.map((color) => `${color} 0%, ${color} 100%`);
    return (
      <article
        className={`grid-entry ${entry.type} ${entry.is_completed ? "completed" : ""} ${fullyElapsed ? "elapsed" : ""}`}
        style={{
          top: `${(start / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`,
          height: `calc(${(duration / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem - ${(trimTop ? EVENT_SEPARATOR_PIXELS : 0) + (trimBottom ? EVENT_SEPARATOR_PIXELS : 0)}px)`,
          transform: trimTop
            ? `translateY(${EVENT_SEPARATOR_PIXELS}px)`
            : undefined,
          ...(entry.type === "event" && mainTag
            ? {
                backgroundColor: mainTag.color,
                color: tagTextColor(mainTag.color),
                borderLeftWidth: 0,
                backgroundImage: `linear-gradient(to right, ${stripStops.join(", ")})`,
                backgroundSize: "100% 3px",
                backgroundPosition: "top",
                backgroundRepeat: "no-repeat",
              }
            : {}),
        }}
        key={entry.id}
        onClick={(event) => {
          event.stopPropagation();
          if (!entry.is_shared) editItem(entry);
        }}
      >
        <strong>{entry.title}</strong>
        <span>
          {entry.type === "event"
            ? `${formatTime(entry.starts_at)}${entry.ends_at ? ` – ${formatTime(entry.ends_at)}` : ""}`
            : "Task"}
        </span>
        {entry.location && <span>{entry.location}</span>}
        {entry.type === "task" && (
          <input
            type="checkbox"
            checked={entry.is_completed}
            onClick={(event) => event.stopPropagation()}
            onChange={() => toggleTask(entry)}
            aria-label={`Complete ${entry.title}`}
          />
        )}
      </article>
    );
  };
  const renderDayEntries = (date: Date) => {
    const dayEntries = weekEntries.filter(
      (entry) =>
        localDateKey(
          getTimelineEntryDate(entry),
        ) === dateKey(date),
    );
    return dayEntries.map((entry, index) => {
      const previous = dayEntries[index - 1];
      const next = dayEntries[index + 1];
      const { trimTop, trimBottom } = getBackToBackTrim(
        entry,
        previous,
        next,
      );
      return renderGridEntry(entry, trimTop, trimBottom);
    });
  };
  const renderElapsedOverlay = (date: Date) => {
    const state = getElapsedOverlayState(date, today, (value) => format(value, "p"));
    if (state.kind === "past") {
      return <div className="day-elapsed-overlay past" aria-hidden="true" />;
    }
    if (state.kind === "none") return null;
    return (
      <div
        className="day-elapsed-overlay today"
        style={{
          height: state.height,
        }}
        aria-hidden="true"
      >
        <span className="current-time-label">{state.label}</span>
      </div>
    );
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
                {renderPreview(date)}
                {renderElapsedOverlay(date)}
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
          onClick={() => deleteItem(editingItem)}
        >
          Delete item
        </button>
      )}
      {showTagEditor && (
        <TagEditorDrawer
          tags={tags}
          error={actionError}
          onClose={() => setShowTagEditor(false)}
          onUpdate={updateTag}
          onUpdateDefault={(tag) => updateDefaultTag(tag.id)}
          onDelete={deleteTag}
        />
      )}
    </main>
  );
}

export default App;
