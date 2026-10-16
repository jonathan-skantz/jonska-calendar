export const queryKeys = {
  currentUser: ["current-user"] as const,
  workspace: ["workspace"] as const,
  transitCard: ["transit-card"] as const,
  transitCardDays: (from: string, to: string) =>
    ["transit-card", from, to] as const,
};
