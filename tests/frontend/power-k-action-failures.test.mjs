import assert from "node:assert/strict";
import { registerHooks } from "./helpers/module-hooks.mjs";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const toasts = [];
globalThis.vectorPowerKExecutionToasts = toasts;
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/propel/toast" && context.parentURL?.endsWith("/power-k/core/execute-action.ts")) {
      const source =
        'export const TOAST_TYPE = {ERROR:"error",SUCCESS:"success"}; export const setToast = toast => globalThis.vectorPowerKExecutionToasts.push(toast);';
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    return nextResolve(specifier, context);
  },
});
const { executePowerKAction, selectPowerKCommand, selectPowerKPageData, copyPowerKCurrentUrl } =
  await import("../../apps/web/core/components/power-k/core/execute-action.ts");
const { ShortcutHandler } = await import("../../apps/web/core/components/power-k/core/shortcut-handler.ts");

await test("command execution starts synchronously and handles thrown and rejected failures", async () => {
  toasts.length = 0;
  const events = [];
  executePowerKAction(() => events.push("action"));
  events.push("dismiss");
  assert.deepEqual(events, ["action", "dismiss"]);
  executePowerKAction(() => {
    throw new Error("private synchronous detail");
  });
  executePowerKAction(() => Promise.reject(new Error("private asynchronous detail")));
  await Promise.resolve();
  assert.equal(toasts.length, 2);
  for (const toast of toasts) {
    assert.equal(toast.type, "error");
    assert.equal(toast.title, "Could not complete command");
    assert.equal(toast.message, "Please try again.");
  }
});

await test("a rejected keyboard command reports failure once and leaves later commands usable", async () => {
  toasts.length = 0;
  let calls = 0;
  let prevented = 0;
  const command = {
    type: "action",
    isVisible: () => true,
    isEnabled: () => true,
    action: () => {
      calls += 1;
      return calls === 1 ? Promise.reject(new Error("private request detail")) : Promise.resolve();
    },
  };
  const registry = { findByKeySequence: () => undefined, findByShortcut: () => command };
  const handler = new ShortcutHandler(
    registry,
    () => ({}),
    () => assert.fail("unexpected palette opening")
  );
  const event = {
    key: "p",
    target: null,
    preventDefault: () => {
      prevented += 1;
    },
  };
  try {
    handler.handleKeyDown(event);
    await Promise.resolve();
    assert.equal(toasts.length, 1);
    handler.handleKeyDown(event);
    await Promise.resolve();
    assert.equal(calls, 2);
    assert.equal(prevented, 2);
    assert.equal(toasts.length, 1);
  } finally {
    handler.destroy();
  }
});

void test("palette selections preserve their dismissal policy and page transitions", async () => {
  const events = [];
  const context = {
    closePalette: () => events.push("close"),
    setActiveCommand: (command) => {
      context.activeCommand = command;
      events.push("command");
    },
    setActivePage: (page) => events.push(page),
  };
  const resetSearch = () => events.push("reset");
  selectPowerKCommand({ type: "unsupported" }, context, resetSearch);
  assert.deepEqual(events, [], "an unrecognized command cannot navigate or dismiss the palette");
  const action = { type: "action", action: () => events.push("action"), closeOnSelect: false };
  selectPowerKCommand(action, context, resetSearch);
  assert.deepEqual(events.splice(0), ["action"]);
  selectPowerKCommand(action, context, resetSearch, true);
  assert.deepEqual(events.splice(0), ["action", "close"]);
  selectPowerKCommand({ ...action, closeOnSelect: true }, context, resetSearch);
  assert.deepEqual(events.splice(0), ["action", "close"]);
  const command = {
    type: "change-page",
    page: "priority",
    closeOnSelect: false,
    onSelect: (data, receivedContext) => {
      assert.equal(receivedContext, context);
      events.push(data);
    },
  };
  selectPowerKCommand(command, context, resetSearch, true);
  assert.deepEqual(events.splice(0), ["command", "priority", "reset"]);
  selectPowerKPageData("urgent", context);
  assert.deepEqual(events.splice(0), ["urgent"]);
  selectPowerKPageData("low", context, true);
  assert.deepEqual(events.splice(0), ["low", "close"]);
  command.closeOnSelect = true;
  selectPowerKPageData("medium", context);
  assert.deepEqual(events.splice(0), ["medium", "close"]);
  context.activeCommand = null;
  selectPowerKPageData("ignored", context);
  assert.deepEqual(events.splice(0), []);
  selectPowerKPageData("ignored", context, true);
  assert.deepEqual(events.splice(0), ["close"]);
  toasts.length = 0;
  context.activeCommand = { ...command, onSelect: () => Promise.reject(new Error("private selection failure")) };
  selectPowerKPageData("high", context);
  await Promise.resolve();
  assert.equal(toasts.length, 1);
  assert.deepEqual(events, ["close"]);
});

void test("copying the current page reports success and clipboard failures without leaking details", async (t) => {
  const { createReactDomTestEnvironment } = await import("./helpers/react-dom.mjs");
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const copied = [];
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: {
      writeText: async (value) => {
        copied.push(value);
      },
    },
  });
  toasts.length = 0;
  await copyPowerKCurrentUrl("Copied page", "Could not copy");
  assert.deepEqual(copied, [window.location.href]);
  assert.deepEqual(toasts, [{ type: "success", title: "Copied page" }]);
  t.mock.method(navigator.clipboard, "writeText", async () => {
    throw new Error("private clipboard detail");
  });
  await copyPowerKCurrentUrl("Copied page", "Could not copy");
  assert.deepEqual(toasts.at(-1), { type: "error", title: "Could not copy" });
});
