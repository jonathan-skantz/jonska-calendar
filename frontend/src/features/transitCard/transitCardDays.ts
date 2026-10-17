import { dateKey } from "../../lib/calendar";
import type {
  HouseholdMember,
  TransitCardDay,
  TransitCardNeed,
} from "../../types";

type TransitCardDayState =
  "past" | "not_reported" | "not_needed" | "maybe" | "resolved" | "conflict";

export type TransitCardMemberAnswer = HouseholdMember & {
  need?: TransitCardNeed;
};

export type TransitCardDayView = {
  date: Date;
  key: string;
  state: TransitCardDayState;
  day?: TransitCardDay;
  // Other household members' answers; need is undefined when not reported.
  others: TransitCardMemberAnswer[];
};

// A missing row means "not reported". Days that have ended are read-only and
// are never shown as missing, but keep their recorded answer for display.
// The state and day describe only the current user's answer; other household
// members' answers are listed separately in others.
export const getTransitCardDayViews = (
  dates: Date[],
  days: TransitCardDay[],
  today: Date,
  userId: string,
  members: HouseholdMember[] = [],
): TransitCardDayView[] => {
  const todayKey = dateKey(today);
  const byDateAndUser = new Map(
    days.map((day) => [`${day.date}/${day.user_id}`, day]),
  );
  const otherMembers = members.filter((member) => member.user_id !== userId);
  return dates.map((date) => {
    const key = dateKey(date);
    const day = byDateAndUser.get(`${key}/${userId}`);
    const others = otherMembers.map((member) => ({
      ...member,
      need: byDateAndUser.get(`${key}/${member.user_id}`)?.need,
    }));
    let state: TransitCardDayState;
    if (key < todayKey) state = "past";
    else if (!day) state = "not_reported";
    else if (day.need === "no") state = "not_needed";
    else if (day.need === "maybe") state = "maybe";
    else state = day.status;
    return { date, key, state, day, others };
  });
};

export type TransitCardPillVariant =
  "pending" | "not_needed" | "maybe" | "resolved" | "conflict";

// Every answer gets a pill, so "no" is never confused with "not reported".
// Only days that ended without an answer get no pill.
export const getTransitCardPillVariant = (
  view: TransitCardDayView,
): TransitCardPillVariant | null => {
  if (view.state === "not_reported") return "pending";
  if (!view.day) return null;
  if (view.day.need === "no") return "not_needed";
  return view.day.need === "maybe" ? "maybe" : view.day.status;
};

// Counts "yes" answers for a day using the current user's live choice, so the
// editor can warn before saving. Maybe never uses up a card.
export const getDayCapacity = (
  view: TransitCardDayView,
  ownNeed: TransitCardNeed | undefined,
  cardCount: number,
) => {
  const yesCount =
    view.others.filter((member) => member.need === "yes").length +
    (ownNeed === "yes" ? 1 : 0);
  return { yesCount, overCapacity: yesCount > cardCount };
};

// Only days that have not ended are submitted. Every one of them gets an
// explicit answer: days without a "yes" or "maybe" are sent as "no", so an
// empty selection is still a report.
export const getTransitCardSubmission = (
  views: TransitCardDayView[],
  answers: Map<string, TransitCardNeed>,
) => {
  const editable = views.filter((view) => view.state !== "past");
  if (editable.length === 0) return null;
  const keysWith = (need: TransitCardNeed) =>
    editable
      .filter((view) => answers.get(view.key) === need)
      .map((view) => view.key);
  return {
    from: editable[0].key,
    to: editable[editable.length - 1].key,
    needed_dates: keysWith("yes"),
    maybe_dates: keysWith("maybe"),
  };
};
