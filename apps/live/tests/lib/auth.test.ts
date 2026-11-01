import { beforeEach, describe, expect, it, vi } from "vitest";
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

describe("authentication input validation", () => {
  beforeEach(() => vi.clearAllMocks());

  it.each([null, 42, "user-id", {}, { id: 42 }, { id: "" }, { id: "user-id", cookie: {} }])(
    "rejects malformed credentials before page or user requests: %j",
    async (token) => {
      await expect(authenticate(JSON.stringify(token))).rejects.toMatchObject({ code: "AUTH_MISSING_CREDENTIALS" });
      expect(fetchDetailsMock).not.toHaveBeenCalled();
      expect(currentUserMock).not.toHaveBeenCalled();
    }
  );
  it.each(["", "workspace_page", "constructor"])(
    "rejects unsupported document type %s before fetching a page",
    async (documentType) => {
      await expect(authenticate(JSON.stringify({ id: "user-id" }), documentType)).rejects.toThrow(
        "Invalid document type"
      );
      expect(fetchDetailsMock).not.toHaveBeenCalled();
      expect(currentUserMock).not.toHaveBeenCalled();
    }
  );
  it("prefers a validated token cookie and still checks the authenticated user identity", async () => {
    currentUserMock.mockResolvedValueOnce({ id: "different-user", display_name: "Other" });
    await expect(authenticate(JSON.stringify({ id: "user-id", cookie: "session=token" }))).rejects.toMatchObject({
      code: "AUTH_USER_MISMATCH",
    });
    expect(currentUserMock).toHaveBeenCalledWith("session=token");
  });
});

const authenticate = (token: string, documentType = "project_page") =>
  onAuthenticate({
    documentName: "page-id.2",
    requestHeaders: new Headers({ cookie: "session=abc" }),
    requestParameters: new URLSearchParams({ documentType, projectId: "project", workspaceSlug: "workspace" }),
    context: { cookie: "", userId: "", documentType: "project_page", projectId: null, workspaceSlug: null },
    token,
  });
