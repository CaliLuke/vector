/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router";
import { SITES_ISSUE_LAYOUTS } from "@plane/constants";
import { publishSettingsOptions } from "@/queries/publishing";
import type { TIssueFilterKeys, TIssueLayout } from "@/types/issue";

export function usePublicIssueView(anchor: string | undefined) {
  const [searchParams, setSearchParams] = useSearchParams();
  const { data: settings } = useQuery(publishSettingsOptions(anchor));
  const layouts = SITES_ISSUE_LAYOUTS.filter(({ key }) => settings?.view_props?.[key]);
  const layout = layouts.find(({ key }) => key === searchParams.get("board"))?.key ?? layouts[0]?.key;
  const filters = {
    state: (searchParams.get("state") ?? "").split(",").filter(Boolean),
    priority: (searchParams.get("priority") ?? "").split(",").filter(Boolean),
    labels: (searchParams.get("labels") ?? "").split(",").filter(Boolean),
  };
  const updateParams = (values: Record<string, string | undefined>) => {
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous);
      for (const [key, value] of Object.entries(values)) {
        if (value) next.set(key, value);
        else next.delete(key);
      }
      return next;
    });
  };
  return {
    layout,
    filters,
    setLayout: (value: TIssueLayout) => updateParams({ board: value }),
    setFilter: (key: TIssueFilterKeys, values: string[]) => updateParams({ [key]: values.join(",") }),
    clearFilters: () => updateParams({ state: undefined, priority: undefined, labels: undefined }),
  };
}
