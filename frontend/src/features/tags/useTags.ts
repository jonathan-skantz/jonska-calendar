import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { queryKeys } from "../../lib/queryKeys";
import type { Tag } from "../../types";

export const useTags = () =>
  useQuery({
    queryKey: queryKeys.tags,
    queryFn: () => api<Tag[]>("/api/tags"),
  });
