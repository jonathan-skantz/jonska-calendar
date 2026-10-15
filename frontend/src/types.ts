export type User = {
  id: string;
  email: string;
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

export type AuthMode = "login" | "register";
