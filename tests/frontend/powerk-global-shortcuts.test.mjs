import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { act, React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createRouterFixture } from "./helpers/router.mjs";

registerWebModuleHooks();
const shortcutModalStub = new URL("./helpers/power-k-shortcuts-modal-stub.mjs", import.meta.url);
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (
      specifier === "./ui/modal/shortcuts-root" &&
      context.parentURL?.endsWith("/core/components/power-k/global-shortcuts.tsx")
    ) {
      return nextResolve(shortcutModalStub.href, context);
    }
    return nextResolve(specifier, context);
  },
});

const { NativeCommandStateProvider, PowerKContext, PowerKRegistryContext } =
  await import("../../apps/web/core/providers/native-command-state.tsx");
const { GlobalShortcutsProvider } = await import("../../apps/web/core/components/power-k/global-shortcuts.tsx");

function dispatchKey(key, options = {}) {
  document.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, ...options }));
}

await test("global shortcuts use registered commands and dispose with their owner", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/acme/browse/ISSUE-1");
  const actions = [];

  function Shell() {
    const powerK = React.useContext(PowerKContext);
    const { setTopNavInputRef } = powerK;
    const { commandRegistry } = React.useContext(PowerKRegistryContext);
    const inputRef = React.useRef(null);
    const [registered, setRegistered] = React.useState(true);
    const [shouldCheckTopNavRef, setShouldCheckTopNavRef] = React.useState(false);
    const [topNavRefMatches, setTopNavRefMatches] = React.useState(false);
    React.useEffect(() => {
      inputRef.current = document.getElementById("top-nav");
      setTopNavInputRef(inputRef);
      return () => setTopNavInputRef(null);
    }, [setTopNavInputRef]);
    React.useEffect(() => {
      if (shouldCheckTopNavRef) {
        setTopNavRefMatches(powerK.topNavInputRef?.current === inputRef.current);
      }
    }, [powerK.topNavInputRef, shouldCheckTopNavRef]);

    const commands = React.useMemo(
      () => [
        {
          id: "work-item-shortcut",
          i18n_title: "Work item shortcut",
          group: "contextual",
          contextType: "work-item",
          type: "action",
          shortcut: "p",
          closeOnSelect: true,
          isVisible: () => true,
          isEnabled: () => true,
          action: (context) => actions.push(context.activeContext),
        },
      ],
      []
    );
    const context = React.useMemo(
      () => ({
        activeCommand: null,
        activeContext: powerK.activeContext,
        shouldShowContextBasedActions: true,
        setShouldShowContextBasedActions() {},
        params: { workItem: "ISSUE-1" },
        router: routerFixture.router,
        closePalette() {},
        setActiveCommand() {},
        setActivePage() {},
      }),
      [powerK.activeContext]
    );
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("input", { id: "top-nav" }),
      React.createElement(
        "output",
        { id: "registry" },
        commandRegistry
          .getAllCommands()
          .map(({ id }) => id)
          .join(",")
      ),
      React.createElement("button", {
        id: "check-top-nav-ref",
        onClick: () => setShouldCheckTopNavRef(true),
      }),
      React.createElement("output", { id: "top-nav-ref" }, String(topNavRefMatches)),
      React.createElement("output", { id: "modal-open" }, String(powerK.isPowerKModalOpen)),
      React.createElement("button", { id: "unregister", onClick: () => setRegistered(false) }),
      registered && React.createElement(GlobalShortcutsProvider, { context, commands })
    );
  }

  t.after(async () => {
    await environment.render(null);
    await environment.unmount();
  });
  await routerFixture.render(
    environment,
    React.createElement(NativeCommandStateProvider, null, React.createElement(Shell))
  );
  await act(async () => environment.container.querySelector("#check-top-nav-ref").click());
  assert.equal(environment.container.querySelector("#top-nav-ref").textContent, "true");
  assert.equal(environment.container.querySelector("#registry").textContent, "work-item-shortcut");
  await act(async () => dispatchKey("p"));
  assert.deepEqual(actions, ["work-item"]);
  await act(async () => dispatchKey("k", { ctrlKey: true }));
  assert.equal(environment.container.querySelector("#modal-open").textContent, "true");
  await act(async () => environment.container.querySelector("#unregister").click());
  assert.equal(environment.container.querySelector("#registry").textContent, "");
  await act(async () => dispatchKey("p"));
  assert.deepEqual(actions, ["work-item"]);
});

await test("AppStateProvider discards native command-palette modal state after sign-out", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const originalStorage = Object.getOwnPropertyDescriptor(globalThis, "localStorage");
  Object.defineProperty(globalThis, "localStorage", { configurable: true, value: window.localStorage });
  const { AppStateProvider } = await import("../../apps/web/core/providers/app-state.tsx");
  const { resetIdentitySession } = await import("../../apps/web/core/queries/identity-session.ts");
  const { useQueryClient } = await import("@tanstack/react-query");
  const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
  t.mock.method(userService, "currentUser", async () => null);
  let client;
  const { CommandPaletteContext } = await import("../../apps/web/core/providers/native-command-state.tsx");

  function Probe() {
    const currentClient = useQueryClient();
    React.useEffect(() => {
      client = currentClient;
    }, [currentClient]);
    const state = React.useContext(CommandPaletteContext);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("button", { id: "open-create", onClick: () => state.toggleCreateIssueModal(true, "CYCLE") }),
      React.createElement("output", { id: "create-open" }, String(state.isCreateIssueModalOpen)),
      React.createElement("output", { id: "create-store" }, state.createIssueStoreType)
    );
  }

  t.after(async () => {
    await environment.render(null);
    client?.clear();
    if (originalStorage) Object.defineProperty(globalThis, "localStorage", originalStorage);
    else delete globalThis.localStorage;
    await environment.unmount();
  });
  await environment.render(React.createElement(AppStateProvider, null, React.createElement(Probe)));
  await act(async () => environment.container.querySelector("#open-create").click());
  assert.equal(environment.container.querySelector("#create-open").textContent, "true");
  assert.equal(environment.container.querySelector("#create-store").textContent, "CYCLE");
  await act(async () => resetIdentitySession(client));
  assert.equal(environment.container.querySelector("#create-open").textContent, "false");
  assert.equal(environment.container.querySelector("#create-store").textContent, "PROJECT");
});
