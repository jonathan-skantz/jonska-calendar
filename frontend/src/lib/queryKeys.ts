export const queryKeys = {
  currentUser: ["current-user"] as const,
  // Everything the calendar shows; invalidating it refreshes all of these.
  workspace: ["workspace"] as const,
  tags: ["workspace", "tags"] as const,
  timelines: ["workspace", "timeline"] as const,
  // Under workspace: subscriptions change which events the calendar shows.
  subscriptions: ["workspace", "subscriptions"] as const,
  feeds: ["feeds"] as const,
  timeline: (from: string, to: string) =>
    ["workspace", "timeline", from, to] as const,
  transitCard: ["transit-card"] as const,
  household: ["household"] as const,
  reminders: ["reminders"] as const,
  eventShares: (eventId: string) => ["event-shares", eventId] as const,
  transitCardDays: (from: string, to: string) =>
    ["transit-card", from, to] as const,
  weather: (lat: number, lon: number) => ["weather", lat, lon] as const,
};
