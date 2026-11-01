import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const targets = new Map();
const toasts = [];
globalThis.vectorLayoutDropTargets = targets;
globalThis.vectorLayoutDropToasts = toasts;
registerHooks({
  resolve(specifier, context, nextResolve) {
    const stubs = {
      "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter": `
        export const draggable = () => () => {};
        export const dropTargetForElements = options => {
          globalThis.vectorLayoutDropTargets.set(options.element, options);
          return () => globalThis.vectorLayoutDropTargets.delete(options.element);
        };`,
      "@atlaskit/pragmatic-drag-and-drop/combine":
        "export const combine = (...cleanups) => () => cleanups.forEach(cleanup => cleanup());",
      "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge": `
        export const attachClosestEdge = data => data;
        export const extractClosestEdge = data => data.edge ?? null;`,
      "@/components/project-states": "export const StateItemTitle = () => null; export const StateUpdate = () => null;",
      "@plane/ui": "export const DropIndicator = () => null;",
      "@plane/propel/toast":
        "export const TOAST_TYPE = {ERROR:'error'}; export const setToast = value => globalThis.vectorLayoutDropToasts.push(value);",
    };
    if (context.parentURL?.endsWith("/calendar/day-tile.tsx")) {
      stubs["./issue-blocks"] = "export const CalendarIssueBlocks = () => null;";
      stubs["@/components/issues/issue-layouts/utils"] = "export const highlightIssueOnDrop = () => {};";
    }
    return specifier in stubs
      ? { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` }
      : nextResolve(specifier, context);
  },
});
const { StateItem } = await import("../../apps/web/core/components/project-states/state-item.tsx");
const { CalendarDayTile } = await import("../../apps/web/core/components/issues/issue-layouts/calendar/day-tile.tsx");

async function drop(target, data, destination = {}) {
  const source = { data, element: document.createElement("div") };
  const allowed = target.canDrop({ source });
  // Exercise the final callback too: permission and data checks must survive
  // even if a previously accepted drag becomes invalid before it is dropped.
  await act(async () => target.onDrop({ source, self: { data: destination } }));
  return allowed;
}

await test("state drops require a current local state and preserve source groups and destination ordering", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const move = t.mock.fn(async () => {});
  const source = { id: "source", group: "started", sequence: 65535 };
  const destination = { id: "destination", group: "completed", sequence: 131070 };
  const groupedStates = {
    started: [source, { id: "other", group: "started", sequence: 131070 }],
    completed: [destination, { id: "next", group: "completed", sequence: 262140 }],
  };
  const props = {
    state: destination,
    groupKey: "completed",
    groupedStates,
    totalStates: 2,
    stateOperationsCallbacks: { moveStatePosition: move },
  };
  await env.render(React.createElement(StateItem, props));
  let target = [...targets.values()][0];
  await act(async () => {
    for (const data of [
      {},
      { id: 1, groupKey: "started" },
      { id: "foreign", groupKey: "started" },
      { id: "source", groupKey: "constructor" },
      { id: "destination", groupKey: "completed" },
    ]) {
      const dragSource = { data, element: document.createElement("div") };
      assert.equal(target.canDrop({ source: dragSource }), false);
      target.onDrop({ source: dragSource, self: { data: {} } });
    }
  });
  assert.equal(move.mock.callCount(), 0);
  assert.equal(
    await drop(target, { id: "source", groupKey: "started" }, { id: "forged", groupKey: "backlog", edge: "bottom" }),
    true
  );
  assert.deepEqual(move.mock.calls[0].arguments, ["source", { id: "source", group: "completed", sequence: 196605 }]);

  await env.render(
    React.createElement(StateItem, { ...props, groupedStates: { ...groupedStates, started: [source] } })
  );
  target = [...targets.values()][0];
  assert.equal(await drop(target, { id: "source", groupKey: "started" }), false);
  await env.render(React.createElement(StateItem, { ...props, disabled: true }));
  target = [...targets.values()][0];
  assert.equal(await drop(target, { id: "source", groupKey: "started" }), false);
  assert.equal(move.mock.callCount(), 1);
});

await test("calendar drops validate current item dates and permissions before moving a known work item", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  toasts.length = 0;
  const move = t.mock.fn(async () => {});
  const date = new Date(2026, 9, 31);
  const issue = { id: "issue", project_id: "p1", target_date: "2026-10-30", start_date: "2026-10-01" };
  const props = {
    date: { date, is_today: false, is_current_month: true },
    displayFilters: {},
    selectedDate: date,
    issues: { issue },
    groupedIssueIds: {},
    canEditProperties: (projectId) => projectId === "p1",
    handleDragAndDrop: move,
  };
  const payload = { id: "issue", date: "2026-10-30" };
  await env.render(React.createElement(CalendarDayTile, props));
  let target = [...targets.values()][0];
  await act(async () => {
    for (const data of [
      {},
      { ...payload, id: "unknown" },
      { ...payload, id: "constructor" },
      { ...payload, date: {} },
      { ...payload, date: "2026-10-29" },
    ]) {
      const dragSource = { data, element: document.createElement("div") };
      assert.equal(target.canDrop({ source: dragSource }), false);
      target.onDrop({ source: dragSource, self: { data: {} } });
    }
  });
  assert.equal(move.mock.callCount(), 0);
  assert.equal(await drop(target, payload, { date: "forged" }), true);
  assert.deepEqual(move.mock.calls[0].arguments, ["issue", "p1", "2026-10-30", "2026-10-31"]);
  await env.render(React.createElement(CalendarDayTile, { ...props, readOnly: true }));
  assert.equal(await drop([...targets.values()][0], payload), false);
  await env.render(React.createElement(CalendarDayTile, { ...props, canEditProperties: () => false }));
  assert.equal(await drop([...targets.values()][0], payload), false);
  await env.render(
    React.createElement(CalendarDayTile, { ...props, issues: { issue: { ...issue, start_date: "2026-11-01" } } })
  );
  await drop([...targets.values()][0], payload);
  assert.match(toasts.at(-1).message, /Due date cannot be before/);
  assert.equal(move.mock.callCount(), 1);
});
