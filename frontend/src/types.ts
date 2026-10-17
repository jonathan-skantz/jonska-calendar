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
