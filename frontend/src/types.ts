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
  deadline_at?: string;
  kind?: "attendable" | "informational";
  location?: string;
  commute_before_seconds?: number;
  commute_after_seconds?: number;
  is_completed: boolean;
  is_shared?: boolean;
  source_title?: string;
  custom_info?: string;
  tags?: Tag[];
  main_tag_id?: string;
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
