import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  registerPermissionUiStubs,
  settleQuery,
} from "./helpers/permission-ui-fixture.mjs";

registerPermissionUiStubs("/info/actors-info.tsx", {
  "@plane/ui": "export const Avatar = () => null;",
});
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { PageNavigationPaneInfoTabActorsInfo } =
  await import("../../apps/web/core/components/pages/navigation-pane/tab-panels/info/actors-info.tsx");

function row(id, displayName) {
  return {
    id: `membership-${id}`,
    role: 15,
    is_active: true,
    member: { id, display_name: displayName, avatar_url: null },
  };
}

await test("page attribution follows roster readiness and workspace scope without confusing owner and creator", async (t) => {
  let resolveAcme;
  let resolveBeta;
  const acme = new Promise((resolve) => {
    resolveAcme = resolve;
  });
  const beta = new Promise((resolve) => {
    resolveBeta = resolve;
  });
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", (slug) => {
    if (slug === "acme") return acme;
    if (slug === "beta") return beta;
    throw new Error(`Unexpected workspace: ${slug}`);
  });
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/",
    userId: "viewer",
    workspaceRole: 15,
  });
  fixture.client.setQueryData(permissionKeys.workspaceMembership("viewer", "beta"), {
    is_active: true,
    role: 15,
    updated_at: "revision-1",
  });
  const page = {
    owned_by: "owner",
    created_by: "creator",
    updated_by: "editor",
    created_at: "2026-10-20T00:00:00Z",
    updated_at: "2026-10-21T00:00:00Z",
  };
  const render = (workspaceSlug) =>
    fixture.render(React.createElement(PageNavigationPaneInfoTabActorsInfo, { page, workspaceSlug }));
  const links = () => [...fixture.environment.container.querySelectorAll("a")];
  await render("acme");
  assert.equal(links().length, 2);
  assert.ok(links().every((link) => !link.textContent.includes("Acme")));
  await act(async () => {
    resolveAcme([row("creator", "Acme creator"), row("editor", "Acme editor"), row("owner", "Different owner")]);
    await settleQuery();
  });
  assert.deepEqual(
    links().map((link) => link.textContent),
    ["Acme editor", "Acme creator"]
  );
  assert.match(links()[1].getAttribute("href"), /\/acme\/profile\/creator\//);
  assert.doesNotMatch(fixture.environment.container.textContent, /Different owner/);

  await render("beta");
  assert.doesNotMatch(fixture.environment.container.textContent, /Acme/);
  await act(async () => {
    resolveBeta([row("creator", "Beta creator")]);
    await settleQuery();
  });
  assert.equal(links()[1].textContent, "Beta creator");
  assert.match(links()[1].getAttribute("href"), /\/beta\/profile\/creator\//);
  assert.doesNotMatch(links()[0].textContent, /Acme editor/);
  assert.notEqual(links()[0].textContent.trim(), "", "missing actors keep the deactivated-user fallback");
});
