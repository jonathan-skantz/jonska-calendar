import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import type { IcsImportResult } from "../../types";

// The server reads floating times and all-day dates in the browser's zone,
// because it cannot know it.
const browserTimeZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone;

export const icsExportUrl = () =>
  `/api/export.ics?time_zone=${encodeURIComponent(browserTimeZone())}`;

export type IcsImportRequest = {
  ics: string;
  // Empty means the user's default tag.
  tagId: string;
  dryRun: boolean;
};

// A dry run only previews the import, so only a real import refreshes the
// calendar.
export const useIcsImport = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ ics, tagId, dryRun }: IcsImportRequest) =>
      api<IcsImportResult>("/api/import/ics", {
        method: "POST",
        body: JSON.stringify({
          ics,
          time_zone: browserTimeZone(),
          tag_id: tagId || undefined,
          dry_run: dryRun,
        }),
      }),
    onSuccess: (_result, { dryRun }) =>
      dryRun ? undefined : invalidateWorkspace(queryClient),
  });
};
