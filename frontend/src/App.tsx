import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { wcagContrast } from "culori";
import type { FormEvent, MouseEvent } from "react";
import { addDays, format, getHours, getMinutes, startOfWeek } from "date-fns";
import "./App.css";

type User = { id: string; email: string };
type Tag = { id: string; name: string; color: string; is_default: boolean };
type TimelineEntry = {
  id: string;
  type: "event" | "task";
  title: string;
  description?: string;
  url?: string;
  starts_at?: string;
  ends_at?: string;
  deadline_at?: string;
  kind?: "attendable" | "informational";
  location?: string;
  is_completed: boolean;
  is_shared?: boolean;
  source_title?: string;
  custom_info?: string;
  tags?: Tag[];
  main_tag_id?: string;
};
type AuthMode = "login" | "register";

const api = async <T,>(path: string, options?: RequestInit): Promise<T> => {
  const response = await fetch(path, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (!response.ok)
    throw new Error((await response.text()) || "Request failed");
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
};

const formatTime = (value?: string) =>
  value ? format(new Date(value), "p") : "";
const formatDate = (value: Date) => format(value, "EEE, MMM d");
const dateKey = (value: Date) => format(value, "yyyy-MM-dd");
const inputDateTime = (value?: string) =>
  value ? format(new Date(value), "yyyy-MM-dd'T'HH:mm") : "";
const startOfCalendarWeek = (value: Date) =>
  startOfWeek(value, { weekStartsOn: 0 });

const localDateTime = (value: FormDataEntryValue | null) => {
  if (typeof value !== "string" || !value) return undefined;
  return new Date(value).toISOString();
};

const localDateKey = (value?: string) =>
  value ? dateKey(new Date(value)) : "";

const minutesIntoDay = (value?: string) => {
  if (!value) return 0;
  const date = new Date(value);
  return getHours(date) * 60 + getMinutes(date);
};

const sameInstant = (left?: string, right?: string) =>
  Boolean(left && right && new Date(left).getTime() === new Date(right).getTime());

const tagTextColor = (hex: string) =>
  wcagContrast(hex, "white") > wcagContrast(hex, "black")
    ? "#ffffff"
    : "#000000";

function App() {
  const queryClient = useQueryClient();
  const currentUserQuery = useQuery({
    queryKey: ["current-user"],
    queryFn: () => api<User>("/api/auth/me"),
    retry: false,
  });
  const workspaceQuery = useQuery({
    queryKey: ["workspace"],
    queryFn: async () => {
      const [timeline, userTags] = await Promise.all([
        api<TimelineEntry[]>("/api/timeline"),
        api<Tag[]>("/api/tags"),
      ]);
      return { entries: timeline, tags: userTags };
    },
    enabled: Boolean(currentUserQuery.data),
  });
  const user = currentUserQuery.data ?? null;
  const entries = workspaceQuery.data?.entries ?? [];
  const tags = workspaceQuery.data?.tags ?? [];
  const loading = currentUserQuery.isLoading || workspaceQuery.isLoading;
  const [error, setError] = useState("");
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showComposer, setShowComposer] = useState(false);
  const [showTagEditor, setShowTagEditor] = useState(false);
  const [composerType, setComposerType] = useState<"event" | "task">("event");
  const [editingItem, setEditingItem] = useState<TimelineEntry | null>(null);
  const [composerTagIds, setComposerTagIds] = useState<string[]>([]);
  const [composerMainTagId, setComposerMainTagId] = useState<string | null>(
    null,
  );
  const [draftEventStart, setDraftEventStart] = useState<string | null>(null);
  const [hiddenTagIds, setHiddenTagIds] = useState<string[]>([]);
  const [today, setToday] = useState(() => new Date());
  const [weekStart, setWeekStart] = useState(() => startOfCalendarWeek(today));
  const [selectedDate, setSelectedDate] = useState(() => dateKey(today));
  const [actionError, setActionError] = useState("");
  const [invalidFields, setInvalidFields] = useState<string[]>([]);

  useEffect(() => {
    const clock = window.setInterval(() => setToday(new Date()), 60_000);
    return () => window.clearInterval(clock);
  }, []);

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
      await queryClient.invalidateQueries({ queryKey: ["current-user"] });
      await queryClient.invalidateQueries({ queryKey: ["workspace"] });
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to sign in",
      );
    }
  };

  const logout = async () => {
    await api("/api/auth/logout", { method: "POST" });
    await queryClient.invalidateQueries({ queryKey: ["current-user"] });
    queryClient.removeQueries({ queryKey: ["workspace"] });
  };

  const toggleTask = async (entry: TimelineEntry) => {
    try {
      await api(`/api/tasks/${entry.id}`, {
        method: "PATCH",
        body: JSON.stringify({ is_completed: !entry.is_completed }),
      });
      await queryClient.invalidateQueries({ queryKey: ["workspace"] });
    } catch (requestError) {
      setActionError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to update task",
      );
    }
  };

  const deleteItem = async (entry: TimelineEntry) => {
    if (!window.confirm(`Delete “${entry.title}”?`)) return;
    try {
      await api(`/api/timeline/${entry.id}`, { method: "DELETE" });
      await queryClient.invalidateQueries({ queryKey: ["workspace"] });
      setShowComposer(false);
      setEditingItem(null);
      setDraftEventStart(null);
    } catch (requestError) {
      setActionError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to delete item",
      );
    }
  };

  const editItem = (entry: TimelineEntry) => {
    setEditingItem(entry);
    setComposerType(entry.type);
    setDraftEventStart(null);
    setActionError("");
    setInvalidFields([]);
    setComposerTagIds(entry.tags?.map((tag) => tag.id) ?? []);
    setComposerMainTagId(entry.main_tag_id ?? entry.tags?.[0]?.id ?? null);
    setShowComposer(true);
  };

  const openNewComposer = (startAt?: Date) => {
    setEditingItem(null);
    setComposerType("event");
    setComposerTagIds(
      tags.filter((tag) => tag.is_default).map((tag) => tag.id),
    );
    setComposerMainTagId(
      tags.find((tag) => tag.is_default)?.id ?? tags[0]?.id ?? null,
    );
    const start = startAt ?? new Date(`${selectedDate}T09:00`);
    setSelectedDate(dateKey(start));
    setDraftEventStart(start.toISOString());
    setActionError("");
    setInvalidFields([]);
    setShowComposer(true);
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
        new Date(start.getTime() + 2 * 60 * 60 * 1000).toISOString(),
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

  const toggleTagVisibility = (tagID: string) => {
    setHiddenTagIds((current) =>
      current.includes(tagID)
        ? current.filter((id) => id !== tagID)
        : [...current, tagID],
    );
  };

  const updateDefaultTag = async (tagID: string) => {
    try {
      await api("/api/tags/default", {
        method: "PUT",
        body: JSON.stringify({ tag_id: tagID }),
      });
      await queryClient.invalidateQueries({ queryKey: ["workspace"] });
    } catch (requestError) {
      setActionError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to set default tag",
      );
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
      await queryClient.invalidateQueries({ queryKey: ["workspace"] });
    } catch (requestError) {
      setActionError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to update tag",
      );
    }
  };

  const deleteTag = async (tag: Tag) => {
    if (!window.confirm(`Delete tag “${tag.name}”?`)) return;
    try {
      await api(`/api/tags/${tag.id}`, { method: "DELETE" });
      setHiddenTagIds((current) => current.filter((id) => id !== tag.id));
      await queryClient.invalidateQueries({ queryKey: ["workspace"] });
    } catch (requestError) {
      setActionError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to delete tag",
      );
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
      await queryClient.invalidateQueries({ queryKey: ["workspace"] });
    } catch (requestError) {
      setActionError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to create tag",
      );
    }
  };

  const createItem = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const invalidNames = Array.from(event.currentTarget.elements)
      .filter(
        (
          control,
        ): control is
          HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement =>
          control instanceof HTMLInputElement ||
          control instanceof HTMLSelectElement ||
          control instanceof HTMLTextAreaElement,
      )
      .filter((control) => !control.checkValidity())
      .map((control) => control.name)
      .filter(Boolean);
    const startsValue = String(
      (
        event.currentTarget.elements.namedItem(
          "starts_at",
        ) as HTMLInputElement | null
      )?.value ?? "",
    );
    const endsValue = String(
      (
        event.currentTarget.elements.namedItem(
          "ends_at",
        ) as HTMLInputElement | null
      )?.value ?? "",
    );
    if (
      composerType === "event" &&
      startsValue &&
      endsValue &&
      new Date(endsValue) <= new Date(startsValue)
    ) {
      invalidNames.push("starts_at", "ends_at");
    }
    if (composerType === "event" && composerTagIds.length === 0)
      invalidNames.push("event-tags");
    if (
      composerType === "event" &&
      (!composerMainTagId || !composerTagIds.includes(composerMainTagId))
    )
      invalidNames.push("main-tag");
    if (invalidNames.length > 0) {
      setInvalidFields([...new Set(invalidNames)]);
      setActionError("Please correct the highlighted fields.");
      return;
    }
    setInvalidFields([]);
    setActionError("");
    const form = new FormData(event.currentTarget);
    const title = String(form.get("title") || "");
    const description = String(form.get("description") || "") || null;
    const url = String(form.get("url") || "") || null;
    const start = localDateTime(form.get("starts_at"));
    const end = localDateTime(form.get("ends_at"));
    const deadline = localDateTime(form.get("deadline_at"));
    const payload =
      composerType === "event"
        ? {
            title,
            description,
            url,
            kind: form.get("kind") || "informational",
            starts_at: start,
            ends_at: end,
            location: form.get("location") || null,
          }
        : { title, description, url, deadline_at: deadline };
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
      await queryClient.invalidateQueries({ queryKey: ["workspace"] });
      setShowComposer(false);
      setEditingItem(null);
      setActionError("");
    } catch (requestError) {
      setActionError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to create item",
      );
    }
  };

  if (loading)
    return <div className="loading-screen">Loading your calendar...</div>;
  if (!user) {
    return (
      <main className="auth-screen">
        <div className="auth-mark">JS / CAL</div>
        <section className="auth-panel">
          <p className="eyebrow">A private family calendar</p>
          <h1>
            Make room for
            <br />
            <em>what matters.</em>
          </h1>
          <p className="auth-copy">
            Your days, shared with the people who belong in them.
          </p>
          <form onSubmit={submitAuth}>
            <label>
              Email
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
              />
            </label>
            <label>
              Password
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
              />
            </label>
            {error && <p className="error">{error}</p>}
            <button className="primary-button" type="submit">
              {authMode === "login" ? "Enter calendar" : "Create account"}{" "}
              <span>↗</span>
            </button>
          </form>
          <button
            className="text-button"
            onClick={() =>
              setAuthMode(authMode === "login" ? "register" : "login")
            }
          >
            {authMode === "login"
              ? "New here? Create an account"
              : "Already have an account? Sign in"}
          </button>
        </section>
        <div className="auth-note">
          Built for close circles
          <br />
          and ordinary days.
        </div>
      </main>
    );
  }

  const dates = Array.from({ length: 7 }, (_, index) =>
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
    const value = entry.type === "event" ? entry.starts_at : entry.deadline_at;
    const inWeek = dates.some((date) => dateKey(date) === localDateKey(value));
    const matchesTags =
      entry.type !== "event" ||
      !entry.tags?.some((tag) => hiddenTagIds.includes(tag.id));
    return inWeek && matchesTags;
  });
  const openNewEventAt = (event: MouseEvent<HTMLDivElement>, date: Date) => {
    const bounds = event.currentTarget.getBoundingClientRect();
    const hour = Math.max(
      0,
      Math.min(
        22,
        Math.round((event.clientY - bounds.top) / (bounds.height / 24)),
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
          top: `${(minutesIntoDay(draftEventStart ?? undefined) / 60) * 2}rem`,
          height: "4rem",
        }}
        aria-hidden="true"
      >
        <strong>New event</strong>
        <span>
          {formatTime(draftEventStart ?? undefined)} –{" "}
          {formatTime(
            new Date(
              draftStartDate.getTime() + 2 * 60 * 60 * 1000,
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
    const start =
      entry.type === "event"
        ? minutesIntoDay(entry.starts_at)
        : minutesIntoDay(entry.deadline_at);
    const end =
      entry.type === "event" ? minutesIntoDay(entry.ends_at) : start + 45;
    const duration = Math.max(
      entry.type === "event" && end > start ? end - start : 45,
      90,
    );
    const gridScale = 2;
    const fullyElapsed =
      entry.type === "event" &&
      Boolean(entry.ends_at && new Date(entry.ends_at).getTime() <= today.getTime());
    const mainTag =
      entry.tags?.find((tag) => tag.id === entry.main_tag_id) ??
      entry.tags?.[0];
    const stripTags =
      entry.type === "event"
        ? [...(entry.tags?.filter((tag) => tag.id !== mainTag?.id) ?? [])].sort(
            (left, right) => left.name.localeCompare(right.name),
          )
        : [];
    const orderedStripTags = mainTag ? [mainTag, ...stripTags] : stripTags;
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
          top: `${(start / 60) * gridScale}rem`,
          height: `calc(${(duration / 60) * gridScale}rem - ${(trimTop ? 1 : 0) + (trimBottom ? 1 : 0)}px)`,
          transform: trimTop ? "translateY(1px)" : undefined,
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
          entry.type === "event" ? entry.starts_at : entry.deadline_at,
        ) === dateKey(date),
    );
    return dayEntries.map((entry, index) => {
      const previous = dayEntries[index - 1];
      const next = dayEntries[index + 1];
      const trimTop =
        entry.type === "event" &&
        previous?.type === "event" &&
        sameInstant(previous.ends_at, entry.starts_at);
      const trimBottom =
        entry.type === "event" &&
        next?.type === "event" &&
        sameInstant(entry.ends_at, next.starts_at);
      return renderGridEntry(entry, trimTop, trimBottom);
    });
  };
  const renderElapsedOverlay = (date: Date) => {
    const currentDateKey = dateKey(today);
    const dayDateKey = dateKey(date);
    if (dayDateKey < currentDateKey) {
      return <div className="day-elapsed-overlay past" aria-hidden="true" />;
    }
    if (dayDateKey !== currentDateKey) return null;
    const currentMinutes = getHours(today) * 60 + getMinutes(today);
    if (currentMinutes === 0) return null;
    return (
      <div
        className="day-elapsed-overlay today"
        style={{ height: `${(currentMinutes / 60) * 2}rem` }}
        aria-hidden="true"
      >
        <span className="current-time-label">{format(today, "p")}</span>
      </div>
    );
  };

  return (
    <main className="app-shell">
      <header className="topbar">
        <div className="brand">
          <span className="brand-symbol">✳</span>
          <span>calendar / home</span>
        </div>
        <div className="topbar-actions">
          <span className="user-email">{user.email}</span>
          <button className="text-button" onClick={logout}>
            Sign out
          </button>
        </div>
      </header>
      <section className="content-grid">
        <div className="timeline-column">
          <div className="section-heading week-heading">
            <div className="week-title">
              <button
                className="week-nav"
                onClick={() => moveWeek(-7)}
                aria-label="Previous week"
              >
                ←
              </button>
              <div>
                <span className="section-kicker">Week view</span>
                <h2>
                  {formatDate(dates[0])} – {formatDate(dates[6])}
                </h2>
              </div>
              <button
                className="week-nav"
                onClick={() => moveWeek(7)}
                aria-label="Next week"
              >
                →
              </button>
            </div>
            <div className="week-actions">
              <span className="item-count">{weekEntries.length} items</span>
              <button
                className="primary-button compact"
                onClick={() => openNewComposer()}
              >
                Add item <span>＋</span>
              </button>
            </div>
          </div>
          {actionError && !showComposer && !showTagEditor && (
            <p className="error action-error">{actionError}</p>
          )}
          <div className="week-calendar" aria-label="Week calendar">
            <div className="week-day-headings">
              <span />
              {dates.map((date) => (
                <div
                  className={`day-heading ${date.getDay() === 0 || date.getDay() === 6 ? "weekend" : ""}`}
                  key={dateKey(date)}
                >
                  <strong>{date.getDate()}</strong>
                  <span>{formatDate(date).split(",")[0]}</span>
                </div>
              ))}
            </div>
            <div className="week-grid">
              <div className="hour-labels">
                {Array.from({ length: 24 }, (_, hour) => (
                  <span key={hour}>{String(hour).padStart(2, "0")}:00</span>
                ))}
              </div>
              {dates.map((date) => (
                <div
                  className={`day-column ${date.getDay() === 0 || date.getDay() === 6 ? "weekend" : ""}`}
                  key={dateKey(date)}
                  onClick={(event) => openNewEventAt(event, date)}
                >
                  {renderDayEntries(date)}
                  {renderPreview(date)}
                  {renderElapsedOverlay(date)}
                </div>
              ))}
            </div>
          </div>
        </div>
        <aside className="side-panel">
          <div className="pulse-panel">
            <span className="pulse-dot" />
            <div>
              <p className="eyebrow">Today</p>
              <strong>
                {openTasks} open task{openTasks === 1 ? "" : "s"}
              </strong>
            </div>
          </div>
          <div className="side-section">
            <div className="section-heading small">
              <h2>Event filters</h2>
              <button
                className="icon-button"
                aria-label="Add tag"
                onClick={createTag}
              >
                ＋
              </button>
            </div>
            <div className="tag-cloud">
              {tags.length ? (
                tags.map((tag) => {
                  const isHidden = hiddenTagIds.includes(tag.id);
                  return (
                    <div className="tag-control" key={tag.id}>
                      <button
                        className={isHidden ? "tag large hidden" : "tag large"}
                        style={{
                          backgroundColor: tag.color,
                          color: tagTextColor(tag.color),
                        }}
                        onClick={() => toggleTagVisibility(tag.id)}
                        aria-label={`${isHidden ? "Show" : "Hide"} ${tag.name} events`}
                      >
                        <span
                          className={`eye-icon ${isHidden ? "closed" : "open"}`}
                          aria-hidden="true"
                        />
                        {tag.name}
                      </button>
                    </div>
                  );
                })
              ) : (
                <span className="muted">No tags yet</span>
              )}
            </div>
            <p className="tag-help">
              Click a tag to hide or show its events. Default:{" "}
              {tags.find((tag) => tag.is_default)?.name ?? "none"}
            </p>
          </div>
          <div className="tag-editor-trigger">
            <button
              className="text-button"
              onClick={() => {
                setActionError("");
                setShowTagEditor(true);
              }}
            >
              Edit
            </button>
          </div>
          <div className="side-section family-note">
            <span className="note-symbol">✳</span>
            <p>
              Shared events stay yours to arrange. Changes you make to their
              appearance are private.
            </p>
          </div>
        </aside>
      </section>
      {showComposer && (
        <div
          className="drawer-backdrop"
          onMouseDown={(event) =>
            event.target === event.currentTarget &&
            (setShowComposer(false), setEditingItem(null))
          }
        >
          <aside
            className="composer drawer"
            key={`${editingItem?.id ?? "new"}-${composerType}`}
          >
            <div className="modal-heading">
              <div>
                <p className="eyebrow">
                  {editingItem ? "Edit item" : "New item"}
                </p>
                <h2>
                  {editingItem ? "Update the details." : "Give it a place."}
                </h2>
              </div>
              <button
                className="icon-button"
                onClick={() => {
                  setShowComposer(false);
                  setEditingItem(null);
                }}
                aria-label="Close"
              >
                ×
              </button>
            </div>
            {!editingItem && (
              <div className="segmented">
                <button
                  type="button"
                  className={composerType === "event" ? "selected" : ""}
                  onClick={() => setComposerType("event")}
                >
                  Event
                </button>
                <button
                  type="button"
                  className={composerType === "task" ? "selected" : ""}
                  onClick={() => setComposerType("task")}
                >
                  Task
                </button>
              </div>
            )}
            <form onSubmit={createItem}>
              <label>
                Title
                <input
                  name="title"
                  defaultValue={editingItem?.title}
                  placeholder={
                    composerType === "event"
                      ? "What is happening?"
                      : "What needs doing?"
                  }
                  required
                  autoFocus
                />
              </label>
              <label>
                Description
                <textarea
                  name="description"
                  defaultValue={editingItem?.description}
                />
              </label>
              <label>
                URL
                <input
                  name="url"
                  type="url"
                  defaultValue={editingItem?.url}
                  placeholder="https://"
                />
              </label>
              {composerType === "event" ? (
                <>
                  <label>
                    Kind
                    <select
                      name="kind"
                      defaultValue={editingItem?.kind ?? "informational"}
                    >
                      <option value="informational">Informational</option>
                      <option value="attendable">Attendable</option>
                    </select>
                  </label>
                  <label>
                    Starts
                    <input
                      name="starts_at"
                      type="datetime-local"
                      defaultValue={
                        inputDateTime(editingItem?.starts_at) ||
                        `${selectedDate}T09:00`
                      }
                      required
                    />
                  </label>
                  <label>
                    Ends
                    <input
                      name="ends_at"
                      type="datetime-local"
                      defaultValue={
                        inputDateTime(editingItem?.ends_at) ||
                        `${selectedDate}T10:00`
                      }
                      required
                    />
                  </label>
                  <label>
                    Location
                    <input
                      name="location"
                      defaultValue={editingItem?.location}
                      placeholder="Home, online, or somewhere else"
                    />
                  </label>
                  <fieldset className="tag-picker">
                    <legend>Event tags</legend>
                    {tags.map((tag) => (
                      <div className="tag-option" key={tag.id}>
                        <button
                          type="button"
                          className={
                            composerMainTagId === tag.id
                              ? "main-tag-button active"
                              : "main-tag-button"
                          }
                          onClick={() => {
                            setComposerMainTagId(tag.id);
                            setComposerTagIds((current) =>
                              current.includes(tag.id) ? current : [...current, tag.id],
                            );
                          }}
                        >
                          main
                        </button>
                        <label>
                          <input
                            type="checkbox"
                            checked={composerTagIds.includes(tag.id)}
                            onChange={() =>
                              setComposerTagIds((current) =>
                                current.includes(tag.id)
                                  ? current.filter((id) => id !== tag.id)
                                  : [...current, tag.id],
                              )
                            }
                          />
                          {tag.name}
                        </label>
                      </div>
                    ))}
                  </fieldset>
                </>
              ) : (
                <label>
                  Deadline
                  <input
                    name="deadline_at"
                    type="datetime-local"
                    defaultValue={
                      inputDateTime(editingItem?.deadline_at) ||
                      inputDateTime(draftEventStart ?? undefined) ||
                      `${selectedDate}T17:00`
                    }
                    required
                  />
                </label>
              )}
              <button className="primary-button" type="submit">
                {editingItem ? "Save changes" : "Save item"} <span>↗</span>
              </button>
            </form>
          </aside>
        </div>
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
        <div
          className="drawer-backdrop"
          onMouseDown={(event) =>
            event.target === event.currentTarget && setShowTagEditor(false)
          }
        >
          <aside className="composer drawer tag-editor">
            <div className="modal-heading">
              <div>
                <p className="eyebrow">Event filters</p>
                <h2>Edit tags.</h2>
              </div>
              <button
                className="icon-button"
                onClick={() => setShowTagEditor(false)}
                aria-label="Close"
              >
                ×
              </button>
            </div>
            {actionError && <p className="error">{actionError}</p>}
            <div className="tag-editor-list">
              {tags.map((tag) => (
                <form
                  className="tag-editor-entry"
                  key={tag.id}
                  onSubmit={(event) => updateTag(event, tag)}
                >
                  <label>
                    Name
                    <input name="name" defaultValue={tag.name} required />
                  </label>
                  <label>
                    Color
                    <input
                      className="tag-color-input"
                      name="color"
                      type="color"
                      defaultValue={tag.color}
                    />
                  </label>
                  <div className="tag-editor-actions">
                    <button className="primary-button" type="submit">
                      Save
                    </button>
                    <button
                      className="text-button"
                      type="button"
                      onClick={() => updateDefaultTag(tag.id)}
                    >
                      {tag.is_default ? "Default tag" : "Make default"}
                    </button>
                    <button
                      className="danger-button"
                      type="button"
                      onClick={() => deleteTag(tag)}
                    >
                      Delete
                    </button>
                  </div>
                </form>
              ))}
            </div>
          </aside>
        </div>
      )}
    </main>
  );
}

export default App;
