import { AxiosHeaders, type AxiosResponse } from "axios";
import { describe, expect, it, vi } from "vitest";
import { ProjectPageService } from "@/services/page/project-page.service";

const service = (collaborationRevision = 2) =>
  new ProjectPageService({
    workspaceSlug: "workspace",
    projectId: "project",
    cookie: "test-session",
    collaborationRevision,
  });

const response = (data: unknown): AxiosResponse<unknown> => ({
  data,
  status: 200,
  statusText: "OK",
  headers: {},
  config: { headers: new AxiosHeaders() },
});

describe("page collaboration generations", () => {
  it("authorizes the canonical page and rejects a replaced document", async () => {
    const page = service();
    const get = vi.spyOn(page, "get").mockResolvedValue(response({ collaboration_revision: 2, name: "Current" }));
    await expect(page.fetchDetails("page-id.2")).resolves.toMatchObject({ name: "Current" });
    expect(get).toHaveBeenCalledWith("/api/workspaces/workspace/projects/project/pages/page-id/", expect.anything());
    get.mockResolvedValue(response({ collaboration_revision: 3 }));
    await expect(page.fetchDetails("page-id.2")).rejects.toMatchObject({ statusCode: 409 });
  });

  it("guards both body and title saves with the admitted generation", async () => {
    const page = service();
    const patch = vi.spyOn(page, "patch").mockResolvedValue(response({}));
    await page.updatePageProperties("page-id.2", { data: { name: "Edited" } });
    await page.updateDescriptionBinary("page-id.2", {
      description_binary: "AA==",
      description_html: "<p>Edited</p>",
      description_json: {},
    });
    expect(patch).toHaveBeenNthCalledWith(
      1,
      "/api/workspaces/workspace/projects/project/pages/page-id/",
      { name: "Edited", collaboration_revision: 2 },
      expect.anything()
    );
    expect(patch).toHaveBeenNthCalledWith(
      2,
      "/api/workspaces/workspace/projects/project/pages/page-id/description/",
      expect.objectContaining({ collaboration_revision: 2, description_html: "<p>Edited</p>" }),
      expect.anything()
    );
  });

  it("reads a legacy document without a generation as generation zero", async () => {
    const page = service(0);
    const get = vi.spyOn(page, "get").mockResolvedValue(response({ name: "Legacy" }));
    await expect(page.fetchDetails("page-id")).resolves.toMatchObject({ name: "Legacy" });
    expect(get).toHaveBeenCalledWith("/api/workspaces/workspace/projects/project/pages/page-id/", expect.anything());
  });

  it.each(["page-id.01", "page-id.-1", "page-id.9007199254740992", "../page-id", ""])(
    "rejects invalid document identity %s before requesting a page",
    async (documentName) => {
      const page = service();
      const get = vi.spyOn(page, "get");
      await expect(page.fetchDetails(documentName)).rejects.toMatchObject({ statusCode: 400 });
      expect(get).not.toHaveBeenCalled();
    }
  );
});
