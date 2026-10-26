import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const sourceModules = {
  "@plane/propel/icons": "packages/propel/src/icons/actions/close-icon.tsx",
  "@plane/propel/tooltip": "packages/propel/src/tooltip/root.tsx",
  "@atlaskit/pragmatic-drag-and-drop/combine":
    "apps/web/node_modules/@atlaskit/pragmatic-drag-and-drop/dist/cjs/entry-point/combine.js",
  "@atlaskit/pragmatic-drag-and-drop/element/adapter":
    "apps/web/node_modules/@atlaskit/pragmatic-drag-and-drop/dist/cjs/entry-point/element/adapter.js",
  "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge":
    "apps/web/node_modules/@atlaskit/pragmatic-drag-and-drop-hitbox/dist/cjs/closest-edge.js",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@/components/project-states") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const StateItemTitle = () => null; export const StateUpdate = () => null;")}`,
      };
    }

    if (specifier === "@plane/ui") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const DropIndicator = () => null; export const AlertModalCore = ({ isOpen, title }) => isOpen ? title : null;")}`,
      };
    }

    if (specifier === "@plane/propel/toast") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = () => {};")}`,
      };
    }

    if (specifier === "@plane/utils/common") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');")}`,
      };
    }
    if (specifier === "@plane/utils/work-item") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const getCurrentStateSequence = () => 0;")}`,
      };
    }

    if (sourceModules[specifier]) {
      return nextResolve(pathToFileURL(path.join(repositoryRoot, sourceModules[specifier])).href, context);
    }

    return nextResolve(specifier, context);
  },
});

test("StateDelete explains disabled states on focus and only opens deletion for enabled states", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  const errors = [];
  const originalError = console.error;
  console.error = (...args) => errors.push(args.join(" "));
  t.after(() => {
    console.error = originalError;
  });

  const { StateDelete } = await import("../../apps/web/core/components/project-states/options/delete.tsx");
  const renderStateDelete = async (state, totalStates) =>
    environment.render(
      React.createElement(StateDelete, {
        key: state.id,
        totalStates,
        state,
        deleteStateCallback: async () => {},
      })
    );

  await renderStateDelete({ id: "state-default", name: "Started", default: true }, 2);
  let button = environment.container.querySelector("button");
  assert.equal(button?.getAttribute("aria-disabled"), "true", "the default state cannot be deleted");
  assert.equal(button?.getAttribute("aria-label"), "Delete Started state");
  assert.equal(button?.disabled, false, "the default state trigger remains focusable");
  button.focus();
  assert.equal(document.activeElement, button);
  await React.act(async () => new Promise((resolve) => setTimeout(resolve, 250)));
  assert.match(document.body.textContent, /Cannot delete the default state\./);
  await React.act(async () => {
    button.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  assert.doesNotMatch(document.body.textContent, /Cannot delete the default state\./);
  button.blur();
  await React.act(async () => button.click());
  assert.equal(environment.container.textContent, "", "activating the default state does not open deletion");

  await renderStateDelete({ id: "state-only", name: "Started", default: false }, 1);
  button = environment.container.querySelector("button");
  assert.equal(button?.getAttribute("aria-disabled"), "true", "a group with one state cannot be left empty");
  button.focus();
  assert.equal(document.activeElement, button);
  await React.act(async () => new Promise((resolve) => setTimeout(resolve, 250)));
  assert.match(document.body.textContent, /Cannot have an empty group\./);
  await React.act(async () => {
    button.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  assert.doesNotMatch(document.body.textContent, /Cannot have an empty group\./);
  await React.act(async () => button.click());
  assert.equal(environment.container.textContent, "", "activating the singleton does not open deletion");

  await renderStateDelete({ id: "state-2", name: "In Progress", default: false }, 2);
  button = environment.container.querySelector("button");
  assert.ok(button);
  assert.equal(button.getAttribute("aria-disabled"), "false");
  assert.equal(button.disabled, false);
  button.focus();
  assert.equal(document.activeElement, button, "the enabled trigger can receive keyboard focus");
  await React.act(async () => button.click());
  assert.equal(environment.container.textContent, "Delete State", "the enabled trigger opens deletion when activated");

  assert.equal(
    errors.some((message) => message.includes("Function components cannot be given refs")),
    false,
    "Tooltip.Trigger must attach its ref to the native button, not CloseIcon"
  );

  assert.equal(button.querySelector("svg")?.nodeName.toLowerCase(), "svg");
});

test("StateItem cleans up drag and drop registrations during StrictMode effect replay", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  const warnings = [];
  const originalWarn = console.warn;
  console.warn = (...args) => warnings.push(args.join(" "));
  t.after(() => {
    console.warn = originalWarn;
  });

  const { StateItem } = await import("../../apps/web/core/components/project-states/state-item.tsx");
  const state = { id: "state-1", name: "Started", group: "started", default: true };
  await environment.render(
    React.createElement(
      React.StrictMode,
      null,
      React.createElement(StateItem, {
        groupKey: "started",
        groupedStates: { started: [state, { id: "state-2" }] },
        totalStates: 2,
        state,
        stateOperationsCallbacks: {
          moveStatePosition: async () => {},
          updateState: async () => {},
          markStateAsDefault: async () => {},
          deleteState: async () => {},
        },
        shouldTrackEvents: false,
      })
    )
  );

  assert.equal(
    warnings.some((message) => message.includes("already registered") && message.includes("same element")),
    false,
    "StrictMode effect replay must unregister the previous draggable and drop target"
  );
});
