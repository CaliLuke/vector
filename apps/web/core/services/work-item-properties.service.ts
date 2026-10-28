import type { TWorkItemPropertyKind } from "@plane/types";
import type { WorkItemType } from "./work-item-types.service";
import { API_BASE_URL } from "@plane/constants";
import { APIService } from "./api.service";
import { authService } from "./auth.service";

type PropertyKind = TWorkItemPropertyKind;
export type PropertyOption = {
  id: string;
  name: string;
  icon: string;
  parent: string;
  level: number;
  next_label: string;
};
export type PropertyConfiguration = {
  active: boolean;
  required: boolean;
  multiple: boolean;
  format: string;
  content: string;
  expression: string;
  result: string;
  default_value: number | string | string[] | null;
  options: PropertyOption[];
  levels: { id: string; name: string; required: boolean }[];
};
type PropertyContext = {
  id: string;
  name: string;
  project_ids: string[];
  type_ids: string[];
  override_keys: string[];
  configuration: PropertyConfiguration;
};
export type PropertyDefinition = {
  id: string;
  name: string;
  title: string;
  description: string;
  icon: string;
  kind: PropertyKind;
  configuration: PropertyConfiguration;
  contexts: PropertyContext[];
};
type PropertyCatalog = {
  enabled: boolean;
  project_count: number;
  types: WorkItemType[];
  revision: number;
  can_manage: boolean;
  properties: { definition: PropertyDefinition; used: boolean }[];
  attachments: { type_id: string; property_ids: string[] }[];
  projects: { id: string; name: string }[];
  members: { id: string; name: string }[];
};
type PropertyRemoval = {
  action: "delete" | "detach";
  revision: number;
  id: string;
  type_id?: string;
  permanent?: boolean;
  review_id?: string;
};
type ReviewedPropertyUpdate = {
  action: "update";
  revision: number;
  definition: PropertyDefinition;
  review_id?: string;
};
export type ReviewedPropertyChange = PropertyRemoval | ReviewedPropertyUpdate;
export type PropertyRemovalReview = { id: string; items: number; types: number; projects: number };
export type FormulaAuthoringInput = {
  revision: number;
  definition: PropertyDefinition;
  type_id: string;
  project_id?: string;
  issue_id?: string;
};
export type FormulaAuthoringResult = {
  valid: boolean;
  validation_error: string;
  active: boolean;
  evaluated_at: string;
  contexts: { id: string; name: string }[];
  result?: { status: "value" | "empty" | "error"; kind: string; value?: string | number | boolean; error: string };
};
export type PropertyMutation =
  | ReviewedPropertyChange
  | { action: "create"; revision: number; definition: PropertyDefinition }
  | { action: "attach"; revision: number; type_id: string; property_ids: string[] };
class WorkItemPropertiesService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  async catalog(slug: string, signal?: AbortSignal): Promise<PropertyCatalog> {
    return (await this.get(`/api/workspaces/${slug}/work-item-properties/`, {}, { signal })).data;
  }
  async previewFormula(
    slug: string,
    input: FormulaAuthoringInput,
    signal?: AbortSignal
  ): Promise<FormulaAuthoringResult> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    return (
      await this.post(`/api/workspaces/${slug}/work-item-properties/formula-preview/`, input, {
        signal,
        headers: { "X-CSRFTOKEN": csrf_token },
      })
    ).data;
  }
  async previewRemoval(
    slug: string,
    input: ReviewedPropertyChange,
    signal?: AbortSignal
  ): Promise<PropertyRemovalReview> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    return (
      await this.post(`/api/workspaces/${slug}/work-item-properties/removal-preview/`, input, {
        signal,
        headers: { "X-CSRFTOKEN": csrf_token },
      })
    ).data;
  }
  async mutate(slug: string, input: PropertyMutation): Promise<PropertyCatalog> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    return (
      await this.post(`/api/workspaces/${slug}/work-item-properties/`, input, {
        headers: { "X-CSRFTOKEN": csrf_token },
      })
    ).data;
  }
}
export const workItemPropertiesService = new WorkItemPropertiesService();
