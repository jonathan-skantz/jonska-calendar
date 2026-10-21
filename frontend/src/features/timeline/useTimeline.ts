import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { addDays, startOfDay } from "date-fns";
import { api } from "../../lib/api";
import { queryKeys } from "../../lib/queryKeys";
import type { TimelineEntry } from "../../types";

// Days loaded on each side of the visible ones, so dragging or a short step
// shows what is there at once.
const BUFFER_DAYS = 7;

export type TimelineRange = { from: string; to: string };

// The range to load for the visible days: local midnights, as instants.
export const timelineRangeFor = (dates: Date[]): TimelineRange => ({
  from: startOfDay(addDays(dates[0], -BUFFER_DAYS)).toISOString(),
  to: startOfDay(
    addDays(dates[dates.length - 1], BUFFER_DAYS + 1),
  ).toISOString(),
});

// Events in the range, each occurrence of a repeating event on its own, and
// all tasks. The previous range stays on screen while the next one loads.
export const useTimeline = ({ from, to }: TimelineRange) =>
  useQuery({
    queryKey: queryKeys.timeline(from, to),
    queryFn: () =>
      api<TimelineEntry[]>(
        `/api/timeline?${new URLSearchParams({ from, to }).toString()}`,
      ),
    placeholderData: keepPreviousData,
  });
