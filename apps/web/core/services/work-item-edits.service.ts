import { API_BASE_URL } from "@plane/constants";
import type { TIssue, TWorkspaceDraftIssue } from "@plane/types";
import { APIService } from "./api.service";
import { authService } from "./auth.service";

export type ItemEditCommand = {
  restore?: { property_id: string; version_id: string };
  expected_source_version?: string;
  mode: "edit" | "draft" | "publish";
  project_id: string;
  item_id: string;
  revision: number;
  changes: Partial<TIssue>;
};
export type ItemEditReview = {
  id: string;
  preserved_properties: number;
  description_html: string;
};
export type ItemEditCommit = ItemEditCommand & { review_id: string };
export type SavedItemEdit = TIssue | TWorkspaceDraftIssue;

class WorkItemEditsService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  async source(
    workspace: string,
    mode: "edit" | "draft",
    itemId: string,
    projectId: string | undefined,
    signal?: AbortSignal
  ): Promise<{ item: SavedItemEdit; source_version: string }> {
    return (
      await this.get(`/api/workspaces/${workspace}/work-item-edits/source/`, {
        params: { mode, item_id: itemId, project_id: projectId },
        signal,
      })
    ).data;
  }
  async preview(
    workspace: string,
    command: ItemEditCommand,
    beforeSend: () => void,
    signal?: AbortSignal
  ): Promise<ItemEditReview> {
    return this.write(workspace, "preview", command, beforeSend, signal);
  }
  async commit(workspace: string, command: ItemEditCommit, beforeSend: () => void): Promise<SavedItemEdit> {
    return this.write(workspace, "commit", command, beforeSend);
  }
  private async write<T>(
    workspace: string,
    action: string,
    command: ItemEditCommand,
    beforeSend: () => void,
    signal?: AbortSignal
  ): Promise<T> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    beforeSend();
    return (
      await this.post(`/api/workspaces/${workspace}/work-item-edits/${action}/`, command, {
        headers: { "X-CSRFTOKEN": csrf_token },
        signal,
      })
    ).data;
  }
}
export const workItemEditsService = new WorkItemEditsService();

const editFields = [
  "project_id",
  "name",
  "description_html",
  "state_id",
  "priority",
  "assignee_ids",
  "label_ids",
  "cycle_id",
  "module_ids",
  "start_date",
  "target_date",
  "estimate_point",
  "parent_id",
  "type_id",
  "custom_properties",
] as const;

/** Form values also carry display and lifecycle metadata; submit only editable fields. */
export function itemEditChanges(payload: Partial<TIssue>): Partial<TIssue> {
  return Object.fromEntries(
    editFields.filter((field) => Object.hasOwn(payload, field)).map((field) => [field, payload[field]])
  );
}
