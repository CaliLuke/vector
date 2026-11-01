import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

function installAnimationFixture(t) {
  // jsdom has no CSS animation engine or canvas renderer.
  const elementPrototype = window.Element.prototype;
  Object.defineProperty(elementPrototype, "getAnimations", { configurable: true, value: () => [] });
  t.after(() => {
    delete elementPrototype.getAnimations;
  });
  t.mock.method(window.HTMLCanvasElement.prototype, "getContext", () => null);
}

function EmptyCommands() {
  return null;
}

void test("the shortcuts dialog focuses search and returns focus to its launcher", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  installAnimationFixture(t);
  const { PowerKRegistryContext } = await import("../../apps/web/core/providers/native-command-state.tsx");
  const { PowerKCommandRegistry } = await import("../../apps/web/core/components/power-k/core/registry.ts");
  const { ShortcutsModal } = await import("../../apps/web/core/components/power-k/ui/modal/shortcuts-root.tsx");

  function Harness() {
    const [open, setOpen] = React.useState(false);
    return React.createElement(
      PowerKRegistryContext.Provider,
      { value: { commandRegistry: new PowerKCommandRegistry() } },
      React.createElement("button", { onClick: () => setOpen(true) }, "Show shortcuts"),
      React.createElement(ShortcutsModal, { isOpen: open, onClose: () => setOpen(false) })
    );
  }
  await environment.render(React.createElement(Harness));
  const launcher = environment.container.querySelector("button");
  launcher.focus();
  await React.act(async () => launcher.click());
  await waitForReactCondition(
    () => document.activeElement?.getAttribute("placeholder") === "Search for shortcuts",
    "search receives dialog focus"
  );
  await React.act(async () =>
    document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
  );
  await waitForReactCondition(() => document.activeElement === launcher, "dialog restores launcher focus");
});

void test("palette reopening cancels stale reset work and a closed palette resets only its current context", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  installAnimationFixture(t);
  const { PowerKContext } = await import("../../apps/web/core/providers/native-command-state.tsx");
  const { ProjectsAppPowerKModalWrapper } = await import("../../apps/web/core/components/power-k/ui/modal/wrapper.tsx");
  const changes = [];
  const powerK = { activePage: null, setActivePage: (value) => changes.push(["page", value]) };
  const context = {
    params: {},
    shouldShowContextBasedActions: false,
    activeContext: null,
    setActiveCommand: (value) => changes.push(["old-command", value]),
    setShouldShowContextBasedActions: (value) => changes.push(["old-context", value]),
  };
  const replacement = {
    ...context,
    setActiveCommand: (value) => changes.push(["new-command", value]),
    setShouldShowContextBasedActions: (value) => changes.push(["new-context", value]),
  };
  const render = (isOpen, currentContext = context) =>
    environment.render(
      React.createElement(
        PowerKContext.Provider,
        { value: powerK },
        React.createElement(ProjectsAppPowerKModalWrapper, {
          isOpen,
          context: currentContext,
          commandsListComponent: EmptyCommands,
          hideFooter: true,
          onClose() {},
        })
      )
    );
  await render(true);
  await waitForReactCondition(
    () =>
      document.activeElement?.getAttribute("cmdk-input") !== null && document.activeElement instanceof HTMLInputElement,
    "palette search receives dialog focus"
  );
  t.mock.timers.enable({ apis: ["setTimeout"] });
  await render(false);
  await React.act(async () => t.mock.timers.tick(100));
  await render(true);
  await React.act(async () => t.mock.timers.tick(200));
  assert.deepEqual(changes, [], "closing and reopening must not reset the live palette");
  await render(false);
  await render(false, replacement);
  await React.act(async () => t.mock.timers.tick(200));
  assert.deepEqual(changes, [
    ["page", null],
    ["new-command", null],
    ["new-context", true],
  ]);
  changes.length = 0;
  await render(true);
  await render(false);
  await environment.render(null);
  await React.act(async () => t.mock.timers.tick(500));
  assert.deepEqual(changes, [], "unmount cancels the pending reset");
});
