/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useIssueFilterPreferences } from "@/hooks/use-issue-filter-preferences";
import { useIssuesActions } from "@/hooks/use-issues-actions";

/** Bind list actions to the same native preference snapshot rendered by the layout. */
export function useIssueLayoutActions(...args: Parameters<typeof useIssueFilterPreferences>) {
  const preferences = useIssueFilterPreferences(...args);
  const actions = useIssuesActions(args[0], preferences.requestInput);
  return { ...preferences, actions };
}
