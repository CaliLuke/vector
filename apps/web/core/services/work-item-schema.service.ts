import { API_BASE_URL } from "@plane/constants";
import { authService } from "./auth.service";
import { APIService } from "./api.service";
import type { PropertyDefinition } from "./work-item-properties.service";

export type ItemSchemaInput = {
  mode: "create" | "edit" | "draft" | "publish";
  project_id?: string;
  issue_id?: string;
  draft_id?: string;
  // Undefined retains a saved type; empty string explicitly clears selection.
  type_id?: string;
  // Undefined retains the canonical parent; empty string explicitly removes it.
  parent_id?: string;
};
export type ResolvedProperty = Omit<PropertyDefinition, "contexts"> & { context_id: string };
type ItemFormulaResult = { kind: string; status: "value" | "empty" | "error"; value: unknown; error: string };
type ItemFormulaPreview = {
  name: string;
  priority: string;
  start_date: string;
  target_date: string;
  estimate_point?: string;
  custom_properties: Record<string, unknown>;
};
export type ItemSchema = {
  stored_values: Record<string, unknown>;
  item_updated_at?: string;
  evaluated_at: string;
  computed: Record<string, ItemFormulaResult>;
  enabled: boolean;
  revision: number;
  type_id: string;
  eligible_type_ids: string[];
  selection_reason: string;
  properties: ResolvedProperty[];
  members: { id: string; name: string; eligible: boolean; reason: string }[];
  releases: { id: string; name: string; eligible: boolean; reason: string }[];
};
class WorkItemSchemaService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  async schema(slug: string, input: ItemSchemaInput, signal?: AbortSignal): Promise<ItemSchema> {
    return (await this.get(`/api/workspaces/${slug}/work-item-schema/`, { params: input, signal })).data;
  }
  async preview(
    slug: string,
    input: ItemSchemaInput,
    values: ItemFormulaPreview,
    signal?: AbortSignal,
    beforeSend?: () => void
  ): Promise<ItemSchema> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    beforeSend?.();
    return (
      await this.post(`/api/workspaces/${slug}/work-item-schema/preview/`, values, {
        params: input,
        signal,
        headers: { "X-CSRFTOKEN": csrf_token },
      })
    ).data;
  }
}
export const workItemSchemaService = new WorkItemSchemaService();
