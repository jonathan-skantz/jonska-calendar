// Joins the class names that are set, so conditional classes don't leave
// stray spaces or empty strings behind.
export const cx = (...names: (string | false | null | undefined)[]) =>
  names.filter(Boolean).join(" ");
