import { API_BASE_URL } from "@plane/constants";
import { APIService } from "./api.service";
import { authService } from "./auth.service";

export type TypeRemovalCommand = { revision: number; type_id: string; replacement_id: string };
type TypeRemovalCommit = TypeRemovalCommand & { review_id: string };
export type TypeRemovalReview = {
  id: string;
  items: number;
  drafts: number;
  projects: number;
  preserved: number;
  invalid_items: number;
  blocked_reason: string;
};
export type ReviewedTypeRemoval = { command: TypeRemovalCommit; impact: TypeRemovalReview };

class WorkItemTypeRemovalService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  async preview(slug: string, command: TypeRemovalCommand, beforeSend: () => void): Promise<TypeRemovalReview> {
    return (await this.send(slug, "preview", command, beforeSend)).data;
  }
  async commit(slug: string, command: TypeRemovalCommit, beforeSend: () => void): Promise<void> {
    await this.send(slug, "commit", command, beforeSend);
  }
  private async send(
    slug: string,
    operation: "preview" | "commit",
    command: TypeRemovalCommand | TypeRemovalCommit,
    beforeSend: () => void
  ) {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    beforeSend();
    return this.post(`/api/workspaces/${slug}/work-item-types/removal/${operation}/`, command, {
      headers: { "X-CSRFTOKEN": csrf_token },
    });
  }
}
export const workItemTypeRemovalService = new WorkItemTypeRemovalService();
