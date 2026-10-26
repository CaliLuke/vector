import { API_BASE_URL } from "@plane/constants";
import { APIService } from "./api.service";
import { authService } from "./auth.service";
import type { WorkItemType } from "./work-item-types.service";

export type HierarchyCatalog = {
  enabled: boolean;
  can_manage: boolean;
  revision: number;
  hierarchy_enabled: boolean;
  default_level: number;
  types: WorkItemType[];
};
export type HierarchyConfiguration = {
  revision: number;
  default_level: number;
  placements: { type_id: string; level: number }[];
};
export type HierarchyCommit = HierarchyConfiguration & { review_id: string; confirm: string };
type HierarchyReview = {
  id: string;
  links: number;
  restricted_links: number;
  pairs: {
    child_id: string;
    child_name: string;
    child_project_id: string;
    parent_id: string;
    parent_name: string;
    parent_project_id: string;
  }[];
};
class WorkItemHierarchyService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  async catalog(slug: string, signal?: AbortSignal): Promise<HierarchyCatalog> {
    return (await this.get(`/api/workspaces/${slug}/work-item-hierarchy/`, {}, { signal })).data;
  }
  preview(slug: string, input: HierarchyConfiguration, signal?: AbortSignal) {
    return this.command<HierarchyReview>(slug, "preview/", input, signal);
  }
  commit(slug: string, input: HierarchyCommit) {
    return this.command<HierarchyCatalog>(slug, "", input);
  }
  private async command<Result>(
    slug: string,
    path: string,
    input: HierarchyConfiguration | HierarchyCommit,
    signal?: AbortSignal
  ): Promise<Result> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    return (
      await this.post(`/api/workspaces/${slug}/work-item-hierarchy/${path}`, input, {
        signal,
        headers: { "X-CSRFTOKEN": csrf_token },
      })
    ).data;
  }
}
export const workItemHierarchyService = new WorkItemHierarchyService();
