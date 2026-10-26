import type { TWorkItemPropertyKind } from "@plane/types";
import { isAxiosError } from "axios";
import { API_BASE_URL } from "@plane/constants";
import { APIService } from "./api.service";
import { authService } from "./auth.service";

export type WorkItemType = {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  active: boolean;
  default: boolean;
  mandatory: boolean;
  level: number;
  usage: number;
};
export type WorkItemTypeIdentity = Pick<
  WorkItemType,
  "id" | "name" | "icon" | "color" | "active" | "default" | "level"
>;
export type WorkItemPropertyIdentity = {
  key: `customproperty_${string}`;
  property_id: string;
  name: string;
  icon: string;
  kind: TWorkItemPropertyKind;
  result_kind: "Text" | "Number" | "Boolean" | "Date" | "";
  level_id: string;
  catalog_id: string;
  multiple: boolean;
  groupable: boolean;
  operators: string[];
  options: { id: string; name: string; icon?: string; parent?: string }[];
};
type WorkItemPropertyIdentityCatalog = { enabled: boolean; revision: number; fields: WorkItemPropertyIdentity[] };
type WorkItemTypeIdentityCatalog = {
  enabled: boolean;
  revision: number;
  types: WorkItemTypeIdentity[];
};
type WorkItemTypeCatalog = {
  enabled: boolean;
  can_manage: boolean;
  revision: number;
  project_count: number;
  types: WorkItemType[];
};
export type ProjectWorkItemType = WorkItemType & {
  imported: boolean;
  project_active: boolean;
  available: boolean;
};
type ProjectTypeCatalog = {
  enabled: boolean;
  can_manage: boolean;
  revision: number;
  project_name: string;
  default_type_id: string;
  types: ProjectWorkItemType[];
};
export type ProjectTypeMutation = {
  revision: number;
  action: "import" | "pause" | "resume" | "remove" | "set_default";
  type_ids: string[];
};
export type TypeMutation =
  | { action: "enable"; revision: number }
  | { action: "delete"; id: string; revision: number }
  | {
      action: "create" | "update";
      type: Pick<WorkItemType, "id" | "name" | "description" | "icon" | "color" | "active" | "mandatory">;
      revision: number;
    };
class WorkItemTypesService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  async catalog(slug: string, signal?: AbortSignal): Promise<WorkItemTypeCatalog> {
    return (await this.get(`/api/workspaces/${slug}/work-item-types/`, {}, { signal })).data;
  }
  async identities(slug: string, projectId?: string, signal?: AbortSignal): Promise<WorkItemTypeIdentityCatalog> {
    return (
      await this.get(`/api/workspaces/${slug}/work-item-type-identities/`, projectId ? { project_id: projectId } : {}, {
        signal,
      })
    ).data;
  }
  async propertyIdentities(
    slug: string,
    projectId?: string,
    signal?: AbortSignal
  ): Promise<WorkItemPropertyIdentityCatalog> {
    return (
      await this.get(
        `/api/workspaces/${slug}/work-item-property-identities/`,
        projectId ? { project_id: projectId } : {},
        { signal }
      )
    ).data;
  }
  async projectCatalog(slug: string, projectId: string, signal?: AbortSignal): Promise<ProjectTypeCatalog> {
    return (await this.get(`/api/workspaces/${slug}/projects/${projectId}/work-item-types/`, {}, { signal })).data;
  }
  async mutateProject(slug: string, projectId: string, mutation: ProjectTypeMutation): Promise<ProjectTypeCatalog> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    return (
      await this.post(`/api/workspaces/${slug}/projects/${projectId}/work-item-types/`, mutation, {
        headers: { "X-CSRFTOKEN": csrf_token },
      })
    ).data;
  }
  async mutate(slug: string, mutation: TypeMutation): Promise<WorkItemTypeCatalog> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    const config = { headers: { "X-CSRFTOKEN": csrf_token } };
    const base = `/api/workspaces/${slug}/work-item-types/`;
    switch (mutation.action) {
      case "enable":
        return (await this.post(`${base}enable/`, { revision: mutation.revision }, config)).data;
      case "delete":
        return (await this.post(`${base}${mutation.id}/delete/`, { revision: mutation.revision }, config)).data;
      case "create":
        return (await this.post(base, { ...mutation.type, revision: mutation.revision }, config)).data;
      case "update":
        return (
          await this.put(`${base}${mutation.type.id}/`, { ...mutation.type, revision: mutation.revision }, config)
        ).data;
      default:
        throw new Error("Unknown work item type action.");
    }
  }
}
export const workItemTypesService = new WorkItemTypesService();

export function typeErrorMessage(error: unknown) {
  if (isAxiosError<{ message?: string }>(error) && error.response?.data.message) return error.response.data.message;
  return error instanceof Error ? error.message : "Could not save work item types. Please try again.";
}
