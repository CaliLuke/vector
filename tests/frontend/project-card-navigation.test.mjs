import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { notifyManager, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { registerProjectCardModuleHooks } from "./helpers/project-card-module-hooks.mjs";

// Keep the card's actual links, buttons, permission branches, and click handlers.
// Isolate its services, imagery, and modal contents from the application singleton.
const state = { navigations: [] };
registerProjectCardModuleHooks({ React, state, showModalContents: true });

const project = { id: "project-id", name: "Go Plane", identifier: "GOPLANE", member_role: 20, members: [], network: 2 };

async function setup(t, overrides = {}) {
  const environment = await createReactDomTestEnvironment();
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  state.navigations = [];
  const { ProjectCard } = await import("../../apps/web/core/components/project/card.tsx");
  const fixture = createRouterFixture("/default/projects/");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    // Unsubscribing does not cancel notifications already queued for React.
    // Drain that queue while the DOM globals still exist.
    await act(async () => new Promise((resolve) => notifyManager.schedule(resolve)));
    await environment.unmount();
    assert.deepEqual(errors, []);
  });
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
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

test("project navigation and settings are independently accessible sibling links", async (t) => {
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

test("nonmembers open the join dialog instead of navigating", async (t) => {
  const container = await setup(t, { member_role: null });
  assert.equal(container.querySelector('[aria-label="Project settings for Go Plane"]'), null);
  await act(async () => container.querySelector('[aria-label="Open project Go Plane"]').click());
  assert.deepEqual(state.navigations, []);
  assert.ok(container.querySelector('[data-modal="JoinProjectModal"]'));
});

test("archived project actions stay keyboard-focusable without navigating", async (t) => {
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
