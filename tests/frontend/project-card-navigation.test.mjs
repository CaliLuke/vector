import { registerHooks } from "node:module";
import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { QueryClientProvider } from "@tanstack/react-query";
import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { registerProjectCardModuleHooks } from "./helpers/project-card-module-hooks.mjs";

// Keep the card's actual links, buttons, permission branches, and click handlers.
// Isolate its services, imagery, and modal contents from the application singleton.
const state = { navigations: [], toasts: [], copy: async () => {}, copyMenu: [] };
globalThis.projectCardCopyFixture = state;
registerProjectCardModuleHooks({ React, state, showModalContents: true });
registerHooks({
  resolve(specifier, context, next) {
    if (context.parentURL?.endsWith("/project/card.tsx")) {
      const replacements = {
        "@plane/utils/string":
          "export const copyUrlToClipboard = path => globalThis.projectCardCopyFixture.copy(path);",
        "@plane/propel/toast":
          "export const TOAST_TYPE={INFO:'info',ERROR:'error'};export const setPromiseToast=()=>{};export const setToast=toast=>globalThis.projectCardCopyFixture.toasts.push(toast);",
        "@plane/ui":
          "export const Avatar=()=>null;export const AvatarGroup=({children})=>children;export const ContextMenu=({items})=>{globalThis.projectCardCopyFixture.copyMenu=items;return null;};",
      };
      if (Object.hasOwn(replacements, specifier))
        return { shortCircuit: true, url: "data:text/javascript," + encodeURIComponent(replacements[specifier]) };
    }
    return next(specifier, context);
  },
});

// Finish shared imports before a test installs DOM globals and scoped mocks.
const { ProjectCard } = await import("../../apps/web/core/components/project/card.tsx");
const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { FavoriteService } = await import("../../apps/web/core/services/favorite/favorite.service.ts");

const project = { id: "project-id", name: "Go Plane", identifier: "GOPLANE", member_role: 20, members: [], network: 2 };

async function setup(t, overrides = {}) {
  const { env: environment, client } = await createQueryDomFixture(t, {
    defaultOptions: { queries: { retry: false, gcTime: Infinity } },
  });
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  t.after(() => assert.deepEqual(errors, []));
  state.navigations = [];
  const fixture = createRouterFixture("/default/projects/");
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", async () => []);
  t.mock.method(FavoriteService.prototype, "getFavorites", async () => []);
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  const memberRole = Object.hasOwn(overrides, "member_role") ? overrides.member_role : project.member_role;
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "default",
    projectRoles: memberRole == null ? {} : { "project-id": memberRole },
  });
  await fixture.render(
    environment,
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(ProjectCard, { project: { ...project, ...overrides } })
    )
  );
  t.after(
    fixture.router.subscribe("onResolved", () => {
      state.navigations.push(fixture.router.state.location.pathname.replace(/\/$/, ""));
    })
  );
  return environment.container;
}

void test("project navigation and settings are independently accessible sibling links", async (t) => {
  const container = await setup(t);
  assert.equal(container.querySelector("a a, a button"), null);
  const projectLink = container.querySelector('[aria-label="Open project Go Plane"]');
  const settingsLink = container.querySelector('[aria-label="Project settings for Go Plane"]');
  assert.ok(projectLink);
  assert.ok(settingsLink);
  await act(async () => settingsLink.click());
  assert.deepEqual(state.navigations, ["/default/settings/projects/project-id"]);
  await act(async () => container.querySelector('[aria-label="Open project Go Plane"]').click());
  assert.deepEqual(state.navigations, ["/default/settings/projects/project-id", "/default/projects/project-id/issues"]);
  const memberArea = [...container.querySelectorAll("a")].find((link) => link.textContent.includes("No Member Yet"));
  assert.ok(memberArea, "informational member area retains card navigation");
  await act(async () => memberArea.click());
  assert.equal(state.navigations.at(-1), "/default/projects/project-id/issues");
  assert.equal(state.navigations.length, 3);
});

void test("nonmembers open the join dialog instead of navigating", async (t) => {
  const container = await setup(t, { member_role: null });
  assert.equal(container.querySelector('[aria-label="Project settings for Go Plane"]'), null);
  await act(async () => container.querySelector('[aria-label="Open project Go Plane"]').click());
  assert.deepEqual(state.navigations, []);
  assert.ok(container.querySelector('[data-modal="JoinProjectModal"]'));
});

void test("archived project actions stay keyboard-focusable without navigating", async (t) => {
  const container = await setup(t, { archived_at: "2026-10-09T00:00:00Z" });
  await act(async () => container.querySelector('[aria-label="Open project Go Plane"]').click());
  assert.deepEqual(state.navigations, []);
  const restore = [...container.querySelectorAll("button")].find((button) => button.textContent.trim() === "Restore");
  assert.ok(restore);
  restore.focus();
  assert.equal(document.activeElement, restore);
  await act(async () => restore.click());
  assert.ok(container.querySelector('[data-modal="ArchiveRestoreProjectModal"]'));
  await act(async () => container.querySelector('[aria-label="Delete project"]').click());
  assert.ok(container.querySelector('[data-modal="DeleteProjectModal"]'));
  assert.deepEqual(state.navigations, []);
});

void test("project card copy handles rejection and waits for acceptance before feedback", async (t) => {
  await setup(t);
  state.toasts = [];
  const paths = [];
  let pending = Promise.withResolvers();
  state.copy = (path) => {
    paths.push(path);
    return pending.promise;
  };
  const copy = state.copyMenu.find((item) => item.key === "copy-link");
  assert.ok(copy);
  await act(async () => copy.action());
  assert.deepEqual(state.toasts, []);
  assert.deepEqual(paths, ["default/projects/project-id/issues"]);
  await act(async () => pending.reject(new Error("clipboard denied")));
  assert.equal(state.toasts.at(-1).type, "error");
  pending = Promise.withResolvers();
  await act(async () => copy.action());
  await act(async () => pending.resolve());
  assert.deepEqual(
    state.toasts.map((toast) => toast.type),
    ["error", "info"]
  );
  assert.deepEqual(state.navigations, []);
});
