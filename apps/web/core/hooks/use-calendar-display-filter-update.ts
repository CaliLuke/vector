/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { EIssueFilterType } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IIssueDisplayFilterOptions } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import type { IssueActions } from "./use-issues-actions";
import type { IProjectIssuePreferencesContext } from "./use-project-issue-preferences-context";

export type TCalendarDisplayFilterUpdate = (
  projectId: string,
  filters: Partial<IIssueDisplayFilterOptions>
) => Promise<void>;

export function useCalendarDisplayFilterUpdate(
  storeType: EIssuesStoreType,
  projectPreferences: IProjectIssuePreferencesContext | undefined,
  calendarFilters: IIssueDisplayFilterOptions["calendar"],
  updateFilters: IssueActions["updateFilters"]
) {
  return useCallback(
    async (targetProjectId: string, filters: Partial<IIssueDisplayFilterOptions>) => {
      try {
        if (storeType === EIssuesStoreType.PROJECT) {
          if (!projectPreferences) throw new Error("Project preferences are unavailable for the active issue list.");
          await projectPreferences.save({ display_filters: filters });
          return;
        }

        if (!updateFilters) throw new Error("Calendar display filters are unavailable for this issue list.");
        await updateFilters(targetProjectId, EIssueFilterType.DISPLAY_FILTERS, {
          ...filters,
          ...(filters.calendar ? { calendar: { ...calendarFilters, ...filters.calendar } } : {}),
        });
      } catch (error) {
        setToast({
          title: "Error!",
          type: TOAST_TYPE.ERROR,
          message: error instanceof Error ? error.message : "Failed to update calendar preferences.",
        });
      }
    },
    [calendarFilters, projectPreferences, storeType, updateFilters]
  );
}
