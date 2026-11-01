import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
// Exercise final geometry under the browser's reduced-motion preference.
function preferReducedMotion() {
  window.matchMedia = (media) => ({
    media,
    matches: media === "(prefers-reduced-motion: reduce)",
    addEventListener() {},
    removeEventListener() {},
  });
}

function mockChartBounds(t, legendHeight = 20) {
  t.mock.method(HTMLElement.prototype, "getBoundingClientRect", function () {
    const height = this.classList.contains("recharts-legend-wrapper") ? legendHeight : 400;
    return { x: 0, y: 0, left: 0, top: 0, right: 640, bottom: height, width: 640, height };
  });
}

const pieTooltipDefaults = { active: true, showTooltip: true, dataKey: "count", cells: [{ key: "done", fill: "red" }] };

const data = [
  { month: "Jan", count: 2 },
  { month: "Feb", count: 5 },
];
const axes = { xAxis: { key: "month", label: "Month" }, yAxis: { key: "count", label: "Work items" } };
const series = { key: "count", label: "Count", fill: "#ff0000", stroke: "#ff0000", stackId: "total" };
const examples = [
  { file: "line", name: "LineChart", props: { lines: [series] } },
  { file: "area", name: "AreaChart", props: { areas: [{ ...series, strokeColor: "#ff0000" }] } },
  { file: "bar", name: "BarChart", props: { bars: [{ ...series, textClassName: "" }] } },
  { file: "scatter", name: "ScatterChart", props: { scatterPoints: [series] } },
];

for (const example of examples) {
  void test(`${example.name} retains axis registration and custom tick identity across rerenders`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    mockChartBounds(t);
    const module = await import(`../../packages/propel/src/charts/${example.file}-chart/root.tsx`);
    const Chart = module[example.name];
    let mounts = 0;
    function Tick({ x, y, payload }) {
      React.useEffect(() => {
        mounts++;
      }, []);
      return React.createElement("text", { x, y }, `Tick:${payload?.value}`);
    }
    const props = { ...axes, ...example.props, data, showTooltip: false, customTicks: { x: Tick } };
    await env.render(React.createElement(Chart, props));
    assert.ok(env.container.querySelector("svg"), "the real responsive chart has measured dimensions");
    assert.match(env.container.textContent, /Month/);
    assert.match(env.container.textContent, /Work items/);
    assert.match(env.container.textContent, /Tick:Feb/);
    const before = mounts;
    assert.ok(before > 0);
    await env.render(React.createElement(Chart, { ...props, className: "updated" }));
    assert.equal(mounts, before, "unchanged custom ticks must not remount when the chart rerenders");
  });
}

void test("shared chart tooltips filter unrelated series and resolve payload-dependent colors", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { CustomTooltip } = await import("../../packages/propel/src/charts/components/tooltip.tsx");
  const payload = [
    { dataKey: "count", value: 5 },
    { dataKey: "comparison", value: 99 },
  ];
  await env.render(
    React.createElement(CustomTooltip, {
      active: true,
      label: "Feb",
      payload,
      itemKeys: ["count"],
      itemLabels: { count: "Count" },
      itemDotColors: (entries) => ({ count: entries[0].value === 5 ? "red" : "blue" }),
    })
  );
  assert.match(env.container.textContent, /FebCount:5/);
  assert.doesNotMatch(env.container.textContent, /99/);
  assert.ok(env.container.querySelector('[style="background-color: red;"]'));
});

void test("chart legends preserve series labels and restore opacity after hovering", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  mockChartBounds(t);
  const { LineChart } = await import("../../packages/propel/src/charts/line-chart/root.tsx");
  await env.render(
    React.createElement(LineChart, {
      ...axes,
      data: [
        { month: "Jan", count: 2, other: 3 },
        { month: "Feb", count: 5, other: 6 },
      ],
      lines: [series, { ...series, key: "other", label: "Other series" }],
      legend: { align: "center", layout: "horizontal", verticalAlign: "bottom" },
      showTooltip: false,
    })
  );
  const legend = env.container.querySelector(".recharts-legend-wrapper");
  assert.ok(legend);
  const item = [...legend.querySelectorAll("button")].find((node) => node.textContent === "Count");
  assert.ok(item);
  assert.match(legend.textContent, /Other series/);
  const lines = () =>
    [...env.container.querySelectorAll(".recharts-line")].map((node) =>
      node.querySelector("path.recharts-line-curve")?.getAttribute("opacity")
    );
  assert.deepEqual(lines(), ["1", "1"]);
  await act(async () => item.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
  assert.deepEqual(lines(), ["1", "0.1"]);
  await act(async () => item.dispatchEvent(new MouseEvent("mouseout", { bubbles: true })));
  assert.deepEqual(lines(), ["1", "1"]);
  assert.equal(item.type, "button");
  await act(async () => item.focus());
  assert.deepEqual(lines(), ["1", "0.1"], "keyboard focus highlights the same series as hover");
  await act(async () => item.blur());
  assert.deepEqual(lines(), ["1", "1"]);
});

void test("bar percentages use numeric series without treating label text as a value", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { barShapeVariants } = await import("../../packages/propel/src/charts/bar-chart/bar.tsx");
  const shape = { x: 0, y: 0, width: 20, height: 80, dataKey: "count", payload: { count: 2, other: 3, label: "text" } };
  const bar = { ...series, textClassName: "", showPercentage: true, showTopBorderRadius: () => true };
  await env.render(React.createElement("svg", null, barShapeVariants.bar(shape, bar, ["count", "other", "label"])));
  assert.match(env.container.textContent, /40%/);
  assert.equal(env.container.querySelector("path").getAttribute("fill"), "#ff0000");
  await env.render(
    React.createElement("svg", null, barShapeVariants.lollipop({ ...shape, payload: {} }, bar, ["count"]))
  );
  assert.match(env.container.textContent, /0%/);
  assert.doesNotMatch(env.container.innerHTML, /NaN|Infinity/);
});

void test("treemap tooltip renders a valid value and rejects incomplete external payloads", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { TreeMapTooltip } = await import("../../packages/propel/src/charts/tree-map/tooltip.tsx");
  await env.render(
    React.createElement(TreeMapTooltip, {
      active: true,
      payload: [{ payload: { name: "Done", value: 12, label: "items", icon: { invalid: true } } }],
    })
  );
  assert.match(env.container.textContent, /Done12 items/);
  await env.render(
    React.createElement(TreeMapTooltip, { active: true, payload: [{ payload: { name: "Missing value" } }] })
  );
  assert.equal(env.container.textContent, "");
  await env.render(React.createElement(TreeMapTooltip, { active: true, payload: [{ payload: [] }] }));
  assert.equal(env.container.textContent, "");
});

void test("pie tooltips select known cells, resolve labels and reject incomplete payloads", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { PieChartTooltip } = await import("../../packages/propel/src/charts/pie-chart/tooltip.tsx");
  const props = {
    active: true,
    showTooltip: true,
    dataKey: "count",
    cells: [{ key: "done", fill: "red" }],
    payload: [{ dataKey: "count", name: "Done", value: 3, payload: { key: "done", payload: { title: "Finished" } } }],
    tooltipLabel: (record) => record.title ?? "Unknown",
  };
  await env.render(React.createElement(PieChartTooltip, props));
  assert.match(env.container.textContent, /FinishedDone:3/);
  assert.ok(env.container.querySelector('[style="background-color: red;"]'));
});

for (const [name, patch] of Object.entries({
  hidden: { showTooltip: false },
  inactive: { active: false },
  empty: { payload: [] },
  malformed: { payload: [{ payload: null }] },
  unknown: { payload: [{ payload: { key: "missing" } }] },
})) {
  void test(`pie tooltips do not render ${name} payloads`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const { PieChartTooltip } = await import("../../packages/propel/src/charts/pie-chart/tooltip.tsx");
    await env.render(
      React.createElement(PieChartTooltip, {
        ...pieTooltipDefaults,
        payload: [{ payload: { key: "done" }, dataKey: "count", value: 3 }],
        ...patch,
      })
    );
    assert.equal(env.container.textContent, "");
  });
}

void test("vertical pie legends preserve names and highlight the keyed slice for keyboard users", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  preferReducedMotion();
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.map(String).join(" ")));
  mockChartBounds(t, 80);
  const { PieChart } = await import("../../packages/propel/src/charts/pie-chart/root.tsx");
  await env.render(
    React.createElement(PieChart, {
      data: [
        { key: "done", name: "Done", count: 3 },
        { key: "open", name: "Open", count: 7 },
      ],
      dataKey: "count",
      cells: [
        { key: "done", fill: "red" },
        { key: "open", fill: "blue" },
      ],
      legend: { layout: "vertical", align: "right", verticalAlign: "middle" },
      showTooltip: false,
    })
  );
  const legend = env.container.querySelector(".recharts-legend-wrapper");
  assert.match(legend.textContent, /Done/);
  assert.match(legend.textContent, /Open/);
  const button = [...legend.querySelectorAll("button")].find((item) => item.textContent === "Done");
  const opacities = () =>
    [...env.container.querySelectorAll(".recharts-pie-sector")].map(
      (sector) => sector.querySelector("path")?.getAttribute("opacity") ?? "1"
    );
  await waitForReactCondition(() => opacities().length === 2, "pie slices finish rendering");
  assert.deepEqual(opacities(), ["1", "1"]);
  await act(async () => button.focus());
  assert.deepEqual(opacities(), ["1", "0.1"]);
  await act(async () => button.click());
  assert.equal(opacities().length, 2, "selecting a legend does not remove a slice");
  await act(async () => button.blur());
  assert.deepEqual(opacities(), ["1", "1"]);
  assert.deepEqual(errors, []);
});

void test("area comparison lines retain their dashed style and finite geometry", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  preferReducedMotion();
  mockChartBounds(t);
  const { AreaChart } = await import("../../packages/propel/src/charts/area-chart/root.tsx");
  await env.render(
    React.createElement(AreaChart, {
      ...axes,
      data,
      areas: [{ ...series, strokeColor: "red", smoothCurves: true, showDot: true }],
      comparisonLine: { strokeColor: "blue", dashedLine: true },
      showTooltip: false,
    })
  );
  const comparison = env.container.querySelector(".recharts-line-curve");
  assert.ok(comparison);
  assert.equal(comparison.getAttribute("stroke"), "blue");
  await waitForReactCondition(
    () => /4(?:px)?[, ]+4(?:px)?/.test(comparison.getAttribute("stroke-dasharray")),
    "comparison line finishes its drawing animation"
  );
  assert.match(comparison.getAttribute("d"), /^M/);
  assert.doesNotMatch(comparison.getAttribute("d"), /NaN|Infinity/);
});

void test("bar color callbacks receive the original row and keep tooltip-enabled charts renderable", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  preferReducedMotion();
  mockChartBounds(t);
  const { BarChart } = await import("../../packages/propel/src/charts/bar-chart/root.tsx");
  await env.render(
    React.createElement(BarChart, {
      ...axes,
      data,
      bars: [{ ...series, fill: (row) => (row.count > 2 ? "blue" : "red") }],
      showTooltip: true,
    })
  );
  await waitForReactCondition(
    () => env.container.querySelectorAll(".recharts-bar-rectangle path").length === 2,
    "bar shapes finish rendering"
  );
  const colors = [...env.container.querySelectorAll(".recharts-bar-rectangle path")].map((path) =>
    path.getAttribute("fill")
  );
  assert.deepEqual(colors, ["red", "blue"]);
  assert.doesNotMatch(env.container.innerHTML, /NaN|Infinity/);
});

for (const tooltipLabel of ["Completed work", undefined]) {
  void test(`pie tooltips retain ${tooltipLabel ? "explicit labels" : "the data key fallback"} without nested metadata`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const { PieChartTooltip } = await import("../../packages/propel/src/charts/pie-chart/tooltip.tsx");
    await env.render(
      React.createElement(PieChartTooltip, {
        ...pieTooltipDefaults,
        tooltipLabel,
        payload: [{ dataKey: "count", name: "Done", value: 0, payload: { key: "done" } }],
      })
    );
    assert.equal(env.container.querySelector("p").textContent, tooltipLabel ?? "count");
    assert.match(env.container.textContent, /Done:0/);
    await env.render(
      React.createElement(PieChartTooltip, {
        ...pieTooltipDefaults,
        tooltipLabel: (record) => (Object.keys(record).length === 0 ? "Legacy data" : "Incorrect"),
        payload: [{ dataKey: "count", name: "Done", value: 0, payload: { key: "done", payload: null } }],
      })
    );
    assert.equal(env.container.querySelector("p").textContent, "Legacy data");
  });
}
