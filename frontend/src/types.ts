export type User = {
  id: string;
  email: string;
  display_name?: string;
};

export type Tag = {
  id: string;
  name: string;
  color: string;
  is_default: boolean;
};

export type TimelineEntry = {
  id: string;
  type: "event" | "task";
  title: string;
  description?: string;
  url?: string;
  starts_at?: string;
  ends_at?: string;
  is_all_day?: boolean;
  start_is_approximate?: boolean;
  end_is_approximate?: boolean;
  deadline_at?: string;
  kind?: "attendable" | "informational";
  // Only on the current user's own events; a recipient never sees it.
  is_skipped?: boolean;
  location?: string;
  gathering_seconds?: number;
  gathering_is_optional?: boolean;
  gathering_label?: string;
  commute_before_seconds?: number;
  commute_after_seconds?: number;
  is_completed: boolean;
  is_shared?: boolean;
  // Owner's display name, only on events shared with the current user.
  shared_by?: string;
  // Number of recipients, only on the current user's own events.
  share_count?: number;
  tags?: Tag[];
  main_tag_id?: string;
  // On each occurrence of a repeating event: the start its rule gives it,
  // which identifies it even when it was moved, and the series' rule.
  occurrence_start?: string;
  recurrence?: Recurrence;
};

export type Weekday = "MO" | "TU" | "WE" | "TH" | "FR" | "SA" | "SU";

// How an event repeats. A monthly rule repeats on the start's day of the
// month, its weekday as the 1st to 4th of the month, or as the last one.
// until (YYYY-MM-DD, inclusive) and count are never both set.
export type Recurrence = {
  frequency: "daily" | "weekly" | "monthly" | "yearly";
  interval: number;
  weekdays?: Weekday[];
  monthly_by?: "day_of_month" | "weekday" | "last_weekday";
  until?: string;
  count?: number;
};

// A reminder belongs to the current user only and is never shared. It carries
// the event fields needed to deliver it without the timeline.
export type Reminder = {
  id: string;
  event_id: string;
  offset_seconds: number;
  remind_at: string;
  // A reminder is listed, and delivered, once for each occurrence of its
  // event; this is the occurrence's start.
  occurrence_start: string;
  notified_at?: string;
  event_title: string;
  event_location?: string;
  event_starts_at: string;
  event_ends_at: string;
  event_is_all_day?: boolean;
  event_is_skipped?: boolean;
  is_shared?: boolean;
};

// The owner's per-recipient values. A missing field means the recipient sees
// the event's own value.
export type ShareFields = {
  title?: string;
  description?: string;
  url?: string;
  location?: string;
};

export type EventShare = ShareFields & {
  id: string;
  recipient_user_id: string;
  recipient_email: string;
  recipient_display_name: string;
};

export type TransitCardNeed = "yes" | "maybe" | "no";

export type TransitCardDay = {
  date: string;
  user_id: string;
  display_name: string;
  need: TransitCardNeed;
  status: "resolved" | "conflict";
};

export type HouseholdMember = {
  user_id: string;
  display_name: string;
};

export type Household = {
  id: string;
  name: string;
  transit_card_count: number;
  members: HouseholdMember[];
};

export type AuthMode = "login" | "register";

export type WeatherCondition =
  "clear" | "partly-cloudy" | "cloudy" | "fog" | "rain" | "snow" | "thunder";

export type WeatherDay = {
  date: string;
  conditions: WeatherCondition[];
  // Local "HH:MM"; absent when the sun does not rise or set that day.
  sunrise?: string;
  sunset?: string;
};

// A date as the quick-add text stated it. The browser resolves it, since only
// it knows today, the time zone and the week start.
export type DraftDate =
  | { kind: "relative_days"; days: number }
  | {
      kind: "weekday";
      weekday:
        | "monday"
        | "tuesday"
        | "wednesday"
        | "thursday"
        | "friday"
        | "saturday"
        | "sunday";
      week_offset?: 0 | 1;
    }
  | { kind: "calendar"; month: number; day: number; year?: number };

// Only the fields the text stated; the event form fills in the rest.
export type EventDraft = {
  title?: string;
  date?: DraftDate;
  end_date?: DraftDate;
  start_time?: string;
  end_time?: string;
  duration_minutes?: number;
  all_day?: boolean;
  start_is_approximate?: boolean;
  end_is_approximate?: boolean;
  location?: string;
  kind?: "attendable" | "informational";
  tag_ids?: string[];
  commute_before_minutes?: number;
  commute_after_minutes?: number;
  gathering_minutes?: number;
  gathering_is_optional?: boolean;
  gathering_label?: string;
  reminder_minutes?: number[];
};

export type ParseEventResponse = {
  draft: EventDraft;
  warnings: string[];
  unresolved: string[];
};

export type AssistantActionKind =
  "create" | "move" | "skip" | "unskip" | "delete";

// "done" ran at once and can be undone; "pending" waits for the user.
export type AssistantActionStatus =
  "done" | "pending" | "applied" | "declined" | "undone";

export type AssistantDecision = "apply" | "decline" | "undo";

// Something the assistant did or asks to do to one of the user's own events.
// starts_at and ends_at are the event's times after the action.
export type AssistantAction = {
  id: string;
  kind: AssistantActionKind;
  status: AssistantActionStatus;
  event_id: string;
  title: string;
  starts_at: string;
  ends_at: string;
  is_all_day: boolean;
  warnings?: string[];
};

// One tool call the assistant made, summarized.
export type AssistantStep = {
  tool: string;
  summary: string;
};

export type AssistantReply = {
  conversation_id: string;
  reply: string;
  actions: AssistantAction[];
  steps: AssistantStep[];
  // The conversation cannot take another message.
  ended: boolean;
};

// What an .ics import did, or would do for a preview. skipped lists the
// events in the file that are not imported, with the reason.
export type IcsImportResult = {
  imported: number;
  skipped: { title: string; reason: string }[];
  warnings: string[];
};
