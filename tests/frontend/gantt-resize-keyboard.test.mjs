import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
globalThis.ganttResizeToasts = [];
registerWebTestModules({
  "@plane/propel/toast":
    'export const TOAST_TYPE={ERROR:"error"}; export const setToast=value=>globalThis.ganttResizeToasts.push(value);',
});
const { GanttChartStateProvider } = await import("../../apps/web/core/components/gantt-chart/contexts/chart-state.tsx");
const { useGanttResizable } =
  await import("../../apps/web/core/components/gantt-chart/helpers/blockResizables/use-gantt-resizable.ts");
const { ChartDraggable } = await import("../../apps/web/core/components/gantt-chart/helpers/draggable.tsx");
const { projectGanttBlocks } = await import("../../apps/web/core/components/gantt-chart/state.ts");
const initialView = {
  key: "week",
  data: { startDate: new Date(2026, 0, 1), endDate: new Date(2026, 0, 14), dayWidth: 60 },
};
const item = { id: "item", name: "Work", start_date: "2026-01-03", target_date: "2026-01-05", project_id: "project" };
async function fixture(t, options = {}) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  globalThis.ganttResizeToasts = [];
  const writes = [];
  const subject = { ...item, ...options.item };
  const state = { viewData: initialView };
  const update = async (data) => {
    writes.push(data);
    await options.update?.(data);
  };
  function Block({ block, containerRef, enabled }) {
    const ref = React.useRef(null);
    const resizing = useGanttResizable(block, ref, containerRef, update);
    return React.createElement(
      "div",
      { ref, style: { marginLeft: block.position.marginLeft, width: block.position.width } },
      React.createElement(ChartDraggable, {
        ...resizing,
        block,
        blockToRender: () => "Work",
        enableBlockLeftResize: enabled,
        enableBlockRightResize: enabled,
        enableBlockMove: enabled,
      })
    );
  }
  function Chart({ enabled = true, viewData = initialView }) {
    const [positions, setPositions] = React.useState({});
    const containerRef = React.useRef(null);
    const blocks = projectGanttBlocks([subject.id], [subject], viewData, positions);
    const context = {
      viewData,
      blocksById: blocks,
      setIsDragging: () => {},
      setBlockPosition: (id, position) => setPositions((old) => ({ ...old, [id]: position })),
    };
    return React.createElement(
      GanttChartStateProvider,
      { value: context },
      React.createElement(
        "div",
        { ref: containerRef },
        React.createElement(Block, { block: blocks.item, containerRef, enabled })
      )
    );
  }
  const render = (changes) => {
    Object.assign(state, changes);
    return env.render(React.createElement(Chart, state));
  };
  await render({});
  return {
    env,
    writes,
    render,
    slider: (side) => env.container.querySelector(`[type="range"][aria-label^="${side}"]`),
  };
}
async function key(slider, value) {
  await act(async () => {
    slider.focus();
    slider.dispatchEvent(new KeyboardEvent("keydown", { key: value, bubbles: true, cancelable: true }));
  });
}
await test("Gantt date handles expose dates, accept keyboard steps and clamp to inclusive endpoints", async (t) => {
  const f = await fixture(t);
  const left = f.slider("Start date");
  assert.equal(left.tabIndex, 0);
  assert.equal(left.value, "2");
  assert.match(left.getAttribute("aria-valuetext"), /2026/);
  await key(left, "ArrowRight");
  assert.deepEqual(f.writes.at(-1), [
    { id: "item", meta: { project_id: "project" }, start_date: "2026-01-04", target_date: "2026-01-05" },
  ]);
  await key(left, "End");
  assert.equal(f.writes.at(-1)[0].start_date, "2026-01-05");
  await key(left, "ArrowUp");
  assert.equal(f.writes.length, 2, "start cannot move past the inclusive due date");
  await key(left, "Home");
  assert.equal(f.writes.at(-1)[0].start_date, "2026-01-01");
  const right = f.slider("Due date");
  await key(right, "ArrowDown");
  assert.equal(f.writes.at(-1)[0].target_date, "2026-01-04");
  await key(right, "Home");
  assert.equal(f.writes.at(-1)[0].target_date, "2026-01-01");
  await key(right, "ArrowLeft");
  assert.equal(f.writes.length, 5);
  await key(right, "End");
  assert.equal(f.writes.at(-1)[0].target_date, "2026-01-14");
  await key(right, "Escape");
  assert.equal(f.writes.length, 6);
  await f.render({ enabled: false });
  assert.equal(f.env.container.querySelector('[type="range"]'), null);
});
await test("keyboard date saves serialize requests, retain failed values for retry and survive range expansion", async (t) => {
  const requests = [];
  const f = await fixture(t, {
    update: () => {
      const pending = Promise.withResolvers();
      requests.push(pending);
      return pending.promise;
    },
  });
  const left = f.slider("Start date");
  await key(left, "ArrowRight");
  assert.equal(left.getAttribute("aria-disabled"), "true");
  await key(left, "ArrowRight");
  assert.equal(requests.length, 1);
  await act(async () => requests[0].reject(new Error("offline")));
  assert.equal(left.value, "2");
  assert.equal(globalThis.ganttResizeToasts.length, 1);
  await key(left, "ArrowRight");
  await f.render({ viewData: { ...initialView, data: { ...initialView.data, startDate: new Date(2025, 11, 31) } } });
  await act(async () => requests[1].resolve());
  assert.equal(f.slider("Start date").value, "4", "saved dates are projected against the current chart origin");
  await key(f.slider("Start date"), "ArrowLeft");
  await f.env.render(null);
  await act(async () => requests[2].reject(new Error("late")));
  assert.equal(globalThis.ganttResizeToasts.length, 1, "unmounted charts suppress late errors");
});
await test("half-block resizing preserves the missing date until that end is edited", async (t) => {
  const f = await fixture(t, { item: { target_date: undefined } });
  await key(f.slider("Start date"), "ArrowLeft");
  assert.equal(f.writes.at(-1)[0].start_date, "2026-01-02");
  assert.equal(f.writes.at(-1)[0].target_date, undefined);
  await key(f.slider("Due date"), "ArrowRight");
  assert.equal(f.writes.at(-1)[0].start_date, "2026-01-02");
  assert.equal(f.writes.at(-1)[0].target_date, "2026-01-03");
});

await test("native range changes and pointer dragging persist the same dates and expose hover feedback", async (t) => {
  const { changeFormControl } = await import("./helpers/native-forms.mjs");
  const { SIDEBAR_WIDTH } = await import("../../apps/web/core/components/gantt-chart/constants.ts");
  const f = await fixture(t);
  const left = f.slider("Start date");
  await act(async () => left.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
  assert.match(f.env.container.textContent, /2026/);
  await act(async () =>
    left.dispatchEvent(new MouseEvent("mouseout", { bubbles: true, relatedTarget: document.body }))
  );
  assert.equal(f.env.container.textContent, "Work");
  await changeFormControl(left, "3");
  assert.equal(f.writes.at(-1)[0].start_date, "2026-01-04");
  await act(async () =>
    left.dispatchEvent(
      new MouseEvent("mousedown", { button: 0, clientX: SIDEBAR_WIDTH + 180, bubbles: true, cancelable: true })
    )
  );
  assert.equal(document.activeElement, left);
  await act(async () => document.dispatchEvent(new MouseEvent("mousemove", { clientX: SIDEBAR_WIDTH + 120 })));
  await act(async () => document.dispatchEvent(new MouseEvent("mouseup")));
  assert.deepEqual(f.writes.at(-1), [
    { id: "item", meta: { project_id: "project" }, start_date: "2026-01-03", target_date: "2026-01-05" },
  ]);
  assert.equal(f.writes.length, 2);
});
