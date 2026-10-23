import { describe, expect, it, vi } from "vitest";
import { onAuthenticate } from "@/lib/auth";

const { currentUserMock } = vi.hoisted(() => ({ currentUserMock: vi.fn() }));

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
      requestHeaders: new Headers({ cookie: "session=abc" }),
      requestParameters: new URLSearchParams("projectId=project&workspaceSlug=workspace"),
      context,
      token: JSON.stringify({ id: "user-id" }),
    });

    expect(currentUserMock).toHaveBeenCalledWith("session=abc");
    expect(context).toMatchObject({
      cookie: "session=abc",
      projectId: "project",
      userId: "user-id",
      workspaceSlug: "workspace",
    });
    expect(result).toEqual({ user: { id: "user-id", name: "Test user" } });
  });
});
