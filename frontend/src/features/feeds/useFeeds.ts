import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useActionMutation } from "../../hooks/useActionMutation";
import { api } from "../../lib/api";
import { browserTimeZone } from "../../lib/calendar";
import { queryKeys } from "../../lib/queryKeys";
import type { Feed } from "../../types";

export const useFeeds = () =>
  useQuery({
    queryKey: queryKeys.feeds,
    queryFn: () => api<Feed[]>("/api/feeds"),
  });

export type FeedDetails = {
  name: string;
  // Empty means every own event.
  tagId: string;
  includeAlarms: boolean;
};

const feedBody = ({ name, tagId, includeAlarms }: FeedDetails) => ({
  name,
  tag_id: tagId || undefined,
  include_alarms: includeAlarms,
});

// The address calendar apps subscribe to, and the webcal:// form that opens
// one of them directly.
export const feedUrl = (feed: Feed, origin = window.location.origin) =>
  origin + feed.path;
export const webcalUrl = (url: string) => url.replace(/^https?:/, "webcal:");

// Each action resolves to an error message, or null when it succeeded.
export const useFeedMutations = () => {
  const queryClient = useQueryClient();
  const run = useActionMutation("Unable to update the feed", () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.feeds }),
  );
  const path = (id: string) => `/api/feeds/${encodeURIComponent(id)}`;

  return {
    // All-day dates in the feed are written in the browser's zone.
    create: (details: FeedDetails) =>
      run({
        path: "/api/feeds",
        method: "POST",
        body: { ...feedBody(details), time_zone: browserTimeZone() },
      }),
    update: (id: string, details: FeedDetails) =>
      run({ path: path(id), method: "PATCH", body: feedBody(details) }),
    resetUrl: (id: string) =>
      run({ path: `${path(id)}/token`, method: "POST" }),
    remove: (id: string) => run({ path: path(id), method: "DELETE" }),
  };
};
