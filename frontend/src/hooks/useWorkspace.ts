import { useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { Tag, TimelineEntry } from "../types";

export const useWorkspace = (enabled: boolean) =>
  useQuery({
    queryKey: ["workspace"],
    queryFn: async () => {
      const [timeline, userTags] = await Promise.all([
        api<TimelineEntry[]>("/api/timeline"),
        api<Tag[]>("/api/tags"),
      ]);
      return { entries: timeline, tags: userTags };
    },
    enabled,
  });
