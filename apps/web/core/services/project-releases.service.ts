import { API_BASE_URL } from "@plane/constants";
import { APIService } from "./api.service";
import { authService } from "./auth.service";

export type ProjectRelease = {
  id: string;
  name: string;
  description: string;
  target_date: string;
  created_at: string;
};
export type ReleaseValues = Pick<ProjectRelease, "name" | "description" | "target_date">;
export type ReleaseCatalog = {
  can_manage: boolean;
  revision: number;
  project_name: string;
  releases: ProjectRelease[];
};
export type ReleaseMutation = {
  revision: number;
  action: "create" | "update" | "delete";
  id: string;
  values: ReleaseValues;
};
class ProjectReleasesService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  async catalog(slug: string, project: string, signal?: AbortSignal): Promise<ReleaseCatalog> {
    return (await this.get(`/api/workspaces/${slug}/projects/${project}/work-item-releases/`, { signal })).data;
  }
  async mutate(slug: string, project: string, input: ReleaseMutation, beforeSend: () => void): Promise<ReleaseCatalog> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    beforeSend();
    return (
      await this.post(`/api/workspaces/${slug}/projects/${project}/work-item-releases/`, input, {
        headers: { "X-CSRFTOKEN": csrf_token },
      })
    ).data;
  }
}
export const projectReleasesService = new ProjectReleasesService();
