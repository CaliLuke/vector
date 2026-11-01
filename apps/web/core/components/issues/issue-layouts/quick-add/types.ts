import type { TIssue } from "@plane/types";

/** Creation accepts a draft; server-owned fields arrive in the saved response. */
export type IssueQuickAddCallback = (
  projectId: string | null | undefined,
  data: Partial<TIssue>
) => Promise<TIssue | undefined>;
