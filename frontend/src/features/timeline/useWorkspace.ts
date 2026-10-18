import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { queryKeys } from "../../lib/queryKeys";
import type { Tag, TimelineEntry } from "../../types";

export type Workspace = { entries: TimelineEntry[]; tags: Tag[] };

export const useWorkspace = () =>
  useQuery({
    queryKey: queryKeys.workspace,
    queryFn: async (): Promise<Workspace> => {
      const [timeline, userTags] = await Promise.all([
        api<TimelineEntry[]>("/api/timeline"),
        api<Tag[]>("/api/tags"),
      ]);
      return { entries: timeline, tags: userTags };
    },
  });
