export const queryKeys = {
  currentUser: ["current-user"] as const,
  workspace: ["workspace"] as const,
  transitCard: ["transit-card"] as const,
  household: ["household"] as const,
  eventShares: (eventId: string) => ["event-shares", eventId] as const,
  transitCardDays: (from: string, to: string) =>
    ["transit-card", from, to] as const,
};
