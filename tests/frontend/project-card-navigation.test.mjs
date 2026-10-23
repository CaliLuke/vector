import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

// Keep the card's actual links, buttons, permission branches, and click handlers.
// Isolate its services, imagery, and modal contents from the application singleton.
const state = { React, navigations: [] };
globalThis.__projectCardTest = state;
const element = "const React = globalThis.__projectCardTest.React;";
const empty = "() => null";
const substitutes = {
  "@plane/hooks": "export const useLocalStorage = () => ({setValue() {}, storedValue:false});",
  "@plane/propel/emoji-icon-picker": `export const Logo = ${empty};`,
  "@plane/propel/icons": `export const LinkIcon = ${empty}, LockIcon = ${empty}, NewTabIcon = ${empty}, TrashIcon = ${empty}, CheckIcon = ${empty};`,
  "@plane/propel/toast":
    'export const setPromiseToast = () => {}; export const setToast = () => {}; export const TOAST_TYPE = {INFO:"info"};',
  "@plane/propel/tooltip": "export const Tooltip = ({children}) => children;",
  "@plane/propel/button": `${element} export const Button = ({children, variant, ...props}) => React.createElement('button', props, children);`,
  "@plane/ui": `${element} export const Avatar = ${empty}, ContextMenu = ${empty};
    export const AvatarGroup = ({children}) => children;
    export const FavoriteStar = ({onClick}) => React.createElement('button', {onClick, 'aria-label':'Favorite project'});`,
  "@plane/utils":
    "export const cn = (...args) => args.filter(x => typeof x === 'string').join(' '); export const getFileURL = x => x; export const renderFormattedDate = x => x; export const copyUrlToClipboard = async () => {};",
  "@/hooks/store/use-member": "export const useMember = () => ({getUserDetails: () => undefined});",
  "@/hooks/store/use-project":
    "export const useProject = () => ({addProjectToFavorites: async () => {}, removeProjectFromFavorites: async () => {}});",
  "@/hooks/store/user": "export const useUserPermissions = () => ({allowPermissions: () => true});",

  "@/hooks/use-platform-os": "export const usePlatformOS = () => ({isMobile:false});",
  "@/components/common/cover-image": `export const CoverImage = ${empty};`,
};
for (const [specifier, name] of [
  ["./delete-project-modal", "DeleteProjectModal"],
  ["./join-project-modal", "JoinProjectModal"],
  ["./archive-restore-modal", "ArchiveRestoreProjectModal"],
]) {
  substitutes[specifier] =
    `${element} export const ${name} = ({isOpen}) => isOpen ? React.createElement('div', {'data-modal':'${name}'}) : null;`;
}
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL?.endsWith("/project/card.tsx") && Object.hasOwn(substitutes, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(substitutes[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

const project = { id: "project-id", name: "Go Plane", identifier: "GOPLANE", member_role: 20, members: [], network: 2 };

async function setup(t, overrides = {}) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  t.after(() => assert.deepEqual(errors, []));
  state.navigations = [];
  const { ProjectCard } = await import("../../apps/web/core/components/project/card.tsx");
  const fixture = createRouterFixture("/default/projects/");
  await fixture.render(environment, React.createElement(ProjectCard, { project: { ...project, ...overrides } }));
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
