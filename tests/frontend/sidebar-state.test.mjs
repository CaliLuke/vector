import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { act, React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const componentStubs = {
  "@plane/propel/icons": `export const PreferencesIcon = () => null;`,
  "@plane/propel/icon-button": `const React = globalThis.sidebarStateReact;
    export const IconButton = ({ icon: Icon, ...props }) => React.createElement("button", props, Icon ? React.createElement(Icon) : null);`,
  "@plane/propel/scrollarea": `const React = globalThis.sidebarStateReact;
    export const ScrollArea = ({ children, rootClassName, viewportClassName }) => React.createElement("div", { className: rootClassName }, React.createElement("div", { className: viewportClassName }, children));`,
  "@/components/navigation/customize-navigation-dialog": `export const CustomizeNavigationDialog = () => null;`,
  "@/components/workspace/edition-badge": `export const WorkspaceEditionBadge = () => null;`,
  "@/components/sidebar/sidebar-toggle-button": `export const AppSidebarToggleButton = () => null;`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    const key =
      specifier === "./sidebar-toggle-button" &&
      context.parentURL?.endsWith("/core/components/sidebar/sidebar-wrapper.tsx")
        ? "@/components/sidebar/sidebar-toggle-button"
        : specifier;
    if (Object.hasOwn(componentStubs, key)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(componentStubs[key])}` };
    }
    return nextResolve(specifier, context);
  },
});

const {
  SidebarStateProvider,
  useExtendedProjectSidebarOpen,
  useExtendedSidebarOpen,
  useIssueDetailSidebarCollapsed,
  useProfileSidebarCollapsed,
  useSidebarActions,
  useSidebarCollapsed,
  useSidebarDropdownOpen,
  useSidebarPeek,
} = await import("../../apps/web/core/providers/sidebar-state.tsx");

async function setup(t) {
  const environment = await createReactDomTestEnvironment();
  const originalStorage = Object.getOwnPropertyDescriptor(globalThis, "localStorage");
  Object.defineProperty(globalThis, "localStorage", { configurable: true, value: window.localStorage });
  t.after(async () => {
    await environment.render(null);
    await environment.unmount();
    if (originalStorage) Object.defineProperty(globalThis, "localStorage", originalStorage);
    else delete globalThis.localStorage;
  });
  return environment;
}

function stateSnapshot(state) {
  return {
    sidebarCollapsed: state.sidebarCollapsed ?? null,
    sidebarPeek: state.sidebarPeek ?? null,
    extendedSidebar: state.isExtendedSidebarOpened ?? null,
    extendedProjectSidebar: state.isExtendedProjectSidebarOpened ?? null,
    profileSidebarCollapsed: state.profileSidebarCollapsed ?? null,
    issueDetailSidebarCollapsed: state.issueDetailSidebarCollapsed ?? null,
    anyDropdownOpen: state.isAnySidebarDropdownOpen ?? null,
  };
}

function Probe({ id }) {
  const actions = useSidebarActions();
  const state = {
    sidebarCollapsed: useSidebarCollapsed(),
    sidebarPeek: useSidebarPeek(),
    isExtendedSidebarOpened: useExtendedSidebarOpen(),
    isExtendedProjectSidebarOpened: useExtendedProjectSidebarOpen(),
    profileSidebarCollapsed: useProfileSidebarCollapsed(),
    issueDetailSidebarCollapsed: useIssueDetailSidebarCollapsed(),
    isAnySidebarDropdownOpen: useSidebarDropdownOpen(),
  };
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("button", { id: `${id}-sidebar`, onClick: () => actions.toggleSidebar(false) }),
    React.createElement("button", { id: `${id}-extended`, onClick: () => actions.toggleExtendedSidebar(true) }),
    React.createElement("button", { id: `${id}-peek`, onClick: () => actions.toggleSidebarPeek(true) }),
    React.createElement("button", { id: `${id}-dropdown`, onClick: () => actions.toggleAnySidebarDropdown(true) }),
    React.createElement("output", { id: `${id}-state` }, JSON.stringify(stateSnapshot(state)))
  );
}

function ActionOnlyProbe({ onRender }) {
  useSidebarActions();
  React.useEffect(onRender);
  return null;
}

function MobileProbe() {
  const sidebarCollapsed = useSidebarCollapsed();
  const { toggleSidebar } = useSidebarActions();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("button", { id: "mobile-open", onClick: () => toggleSidebar(false) }),
    React.createElement("output", { id: "mobile-state" }, String(sidebarCollapsed))
  );
}

const click = async (environment, selector) => act(async () => environment.container.querySelector(selector).click());

await test("sidebar state is shared, persisted, and transient values reset on a session-keyed remount", async (t) => {
  const environment = await setup(t);
  let actionOnlyRenders = 0;
  const onActionOnlyRender = () => {
    actionOnlyRenders += 1;
  };

  function App({ generation }) {
    return React.createElement(
      SidebarStateProvider,
      { key: generation },
      React.createElement(
        React.Fragment,
        null,
        React.createElement(Probe, { id: "first" }),
        React.createElement(Probe, { id: "second" }),
        React.createElement(ActionOnlyProbe, { onRender: onActionOnlyRender })
      )
    );
  }

  await environment.render(React.createElement(App, { generation: 1 }));
  assert.equal(actionOnlyRenders, 1);
  await click(environment, "#first-sidebar");
  await click(environment, "#first-extended");
  await click(environment, "#first-peek");
  await click(environment, "#first-dropdown");
  assert.equal(actionOnlyRenders, 1, "action-only consumers do not rerender when sidebar values change");

  const sharedState = JSON.parse(environment.container.querySelector("#second-state").textContent);
  assert.equal(sharedState.sidebarCollapsed, false);
  assert.equal(sharedState.extendedSidebar, true);
  assert.equal(sharedState.sidebarPeek, true);
  assert.equal(sharedState.anyDropdownOpen, true);
  assert.equal(localStorage.getItem("app_sidebar_collapsed"), "false");
  assert.equal(localStorage.getItem("extended_sidebar_collapsed"), "true");
  assert.equal(localStorage.getItem("sidebar_peek"), null);

  await environment.render(React.createElement(App, { generation: 2 }));
  assert.equal(actionOnlyRenders, 2, "a session-keyed provider remount creates the new consumer tree");
  const remountedState = JSON.parse(environment.container.querySelector("#second-state").textContent);
  assert.equal(remountedState.sidebarCollapsed, false);
  assert.equal(remountedState.extendedSidebar, true);
  assert.equal(remountedState.sidebarPeek, null);
  assert.equal(remountedState.anyDropdownOpen, null);
});

await test("persisted sidebar choices load after a fresh provider mount", async (t) => {
  const environment = await setup(t);
  localStorage.setItem("app_sidebar_collapsed", "true");
  localStorage.setItem("extended_sidebar_collapsed", "false");
  localStorage.setItem("extended_project_sidebar_collapsed", "true");
  localStorage.setItem("profile_sidebar_collapsed", "false");
  localStorage.setItem("issue_detail_sidebar_collapsed", "true");

  await environment.render(
    React.createElement(SidebarStateProvider, null, React.createElement(Probe, { id: "fresh" }))
  );
  const persistedState = JSON.parse(environment.container.querySelector("#fresh-state").textContent);
  assert.deepEqual(persistedState, {
    sidebarCollapsed: true,
    sidebarPeek: null,
    extendedSidebar: false,
    extendedProjectSidebar: true,
    profileSidebarCollapsed: false,
    issueDetailSidebarCollapsed: true,
    anyDropdownOpen: null,
  });
});

async function verifyMobileSidebar(t, savedCollapsed) {
  const environment = await setup(t);
  globalThis.sidebarStateReact = React;
  const testWindow = environment.container.ownerDocument.defaultView;
  const originalWidth = testWindow.innerWidth;
  testWindow.innerWidth = 375;
  if (savedCollapsed !== null) localStorage.setItem("app_sidebar_collapsed", savedCollapsed);
  const { SidebarWrapper } = await import("../../apps/web/core/components/sidebar/sidebar-wrapper.tsx");
  t.after(() => {
    testWindow.innerWidth = originalWidth;
    delete globalThis.sidebarStateReact;
  });

  await environment.render(
    React.createElement(
      React.StrictMode,
      null,
      React.createElement(
        SidebarStateProvider,
        null,
        React.createElement(SidebarWrapper, { title: "Projects" }, React.createElement(MobileProbe)),
        React.createElement("button", { id: "outside" })
      )
    )
  );
  assert.equal(environment.container.querySelector("#mobile-state").textContent, "true");
  await click(environment, "#mobile-open");
  assert.equal(environment.container.querySelector("#mobile-state").textContent, "false");
  await act(async () =>
    environment.container.querySelector("#outside").dispatchEvent(new MouseEvent("mousedown", { bubbles: true }))
  );
  assert.equal(environment.container.querySelector("#mobile-state").textContent, "true");
}

await test("mobile sidebar starts closed without a saved preference", (t) => verifyMobileSidebar(t, null));
await test("mobile sidebar remains closed after hydrating a saved-open preference", (t) =>
  verifyMobileSidebar(t, "false"));
await test("mobile sidebar preserves a saved-closed preference", (t) => verifyMobileSidebar(t, "true"));

await test("changing peek state does not rerender an unrelated sidebar selector", async (t) => {
  const environment = await setup(t);
  let collapsedRenders = 0;
  let actions;
  function CollapsedOnly() {
    collapsedRenders++;
    return React.createElement("output", null, String(useSidebarCollapsed()));
  }
  function Controls() {
    const current = useSidebarActions();
    React.useLayoutEffect(() => {
      actions = current;
    });
    return React.createElement("output", { "data-peek": "" }, String(useSidebarPeek()));
  }
  await environment.render(
    React.createElement(SidebarStateProvider, null, React.createElement(CollapsedOnly), React.createElement(Controls))
  );
  const before = collapsedRenders;
  await act(async () => actions.toggleSidebarPeek(true));
  assert.equal(environment.container.querySelector("[data-peek]").textContent, "true");
  assert.equal(collapsedRenders, before);
});
