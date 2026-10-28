import { describe, expect, it, vi } from "vitest";
import { onAuthenticate } from "@/lib/auth";

const { currentUserMock, fetchDetailsMock } = vi.hoisted(() => ({
  currentUserMock: vi.fn(),
  fetchDetailsMock: vi.fn(),
}));
vi.mock("@/services/page/handler", () => ({ getPageService: () => ({ fetchDetails: fetchDetailsMock }) }));

vi.mock("@/services/user.service", () => ({
  UserService: class {
    currentUser = currentUserMock;
  },
}));

describe("Hocuspocus authentication headers", () => {
  it("reads the cookie from web-standard request Headers when the token has no cookie", async () => {
    currentUserMock.mockResolvedValueOnce({ id: "user-id", display_name: "Test user" });
    const context = {
      projectId: null,
      cookie: "",
      documentType: "project_page" as const,
      workspaceSlug: null,
      userId: "",
    };

    const result = await onAuthenticate({
      documentName: "page-id.2",
      requestHeaders: new Headers({ cookie: "session=abc" }),
      requestParameters: new URLSearchParams("projectId=project&workspaceSlug=workspace&documentType=project_page"),
      context,
      token: JSON.stringify({ id: "user-id" }),
    });

    expect(currentUserMock).toHaveBeenCalledWith("session=abc");
    expect(fetchDetailsMock).toHaveBeenCalledWith("page-id.2");
    expect(context).toMatchObject({
      cookie: "session=abc",
      collaborationRevision: 2,
      projectId: "project",
      userId: "user-id",
      workspaceSlug: "workspace",
    });
    expect(result).toEqual({ user: { id: "user-id", name: "Test user" } });
  });
});
