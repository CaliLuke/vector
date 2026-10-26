import type { QueryKey } from "@tanstack/react-query";

const workspaceQueryPrefixes = new Set([
  "workspace",
  "peek-issue",
  "ISSUE_DETAIL",
  "ISSUE_DETAILS",
  "ARCHIVED_ISSUE_DETAIL",
  "ISSUE_PARENT_CHILD_ISSUES",
  "PROJECT_ISSUES",
  "CYCLE_ISSUES",
  "CYCLE_ISSUES_WITH_PARAMS",
  "MODULE_ISSUES",
  "PROJECT_VIEW_ISSUES",
  "ARCHIVED_ISSUES",
  "WORKSPACE_GLOBAL_VIEW_ISSUES",
]);

/** Schema migration changes existing issue records as well as configuration. */
export function isTypeRemovalQuery(key: QueryKey, slug: string, userId: string | undefined) {
  if (key[0] === "workspace-drafts") return key[1] === userId && key[3] === slug;
  return typeof key[0] === "string" && workspaceQueryPrefixes.has(key[0]) && key[1] === slug;
}
