import { API_BASE_URL } from "@plane/constants";
import { APIService } from "./api.service";

type PropertyHistoryField = {
  property_id: string;
  title: string;
  latest_at: string;
  version_count: number;
};
export type PropertyVersion = {
  id: string;
  property_id: string;
  property_title: string;
  source: "change" | "baseline" | "preservation" | "restore";
  created_at: string;
  actor: { id: string; display_name: string } | null;
};
type PropertyVersionPage = { results: PropertyVersion[]; next_cursor: string | null };
export type PropertyVersionDetail = PropertyVersion & { value: string | null };
export type PropertyHistoryTarget = { workspaceSlug: string; projectId: string; issueId: string };

class WorkItemPropertyHistoryService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  private path(scope: PropertyHistoryTarget) {
    return `/api/workspaces/${scope.workspaceSlug}/projects/${scope.projectId}/work-items/${scope.issueId}/`;
  }
  async catalog(scope: PropertyHistoryTarget, signal?: AbortSignal): Promise<PropertyHistoryField[]> {
    return (await this.get(`${this.path(scope)}property-history/`, { signal })).data;
  }
  async versions(
    scope: PropertyHistoryTarget,
    propertyId: string,
    before: string | null,
    signal?: AbortSignal
  ): Promise<PropertyVersionPage> {
    return (
      await this.get(`${this.path(scope)}properties/${propertyId}/versions/`, {
        params: { before: before ?? undefined },
        signal,
      })
    ).data;
  }
  async detail(
    scope: PropertyHistoryTarget,
    propertyId: string,
    versionId: string,
    signal?: AbortSignal
  ): Promise<PropertyVersionDetail> {
    return (await this.get(`${this.path(scope)}properties/${propertyId}/versions/${versionId}/`, { signal })).data;
  }
}
export const workItemPropertyHistoryService = new WorkItemPropertyHistoryService();
