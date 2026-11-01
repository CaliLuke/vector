import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { SingleProgressStats } = await import("../../apps/web/core/components/core/sidebar/single-progress-stats.tsx");
const { KanbanQuickAddIssueButton } =
  await import("../../apps/web/core/components/issues/issue-layouts/quick-add/button/kanban.tsx");
const { EmptySpaceItem } = await import("../../apps/web/core/components/ui/empty-space.tsx");
const { GanttChartHeader } = await import("../../apps/web/core/components/gantt-chart/chart/header.tsx");
function Icon() {
  return null;
}

void test("progress actions expose selection, avoid submitting forms, and display finite empty totals", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let clicks = 0;
  const submit = t.mock.fn((event) => event.preventDefault());
  await env.render(
    React.createElement(
      "form",
      { onSubmit: submit },
      React.createElement(SingleProgressStats, {
        title: "Completed",
        completed: 1,
        total: 0,
        selected: true,
        onClick: () => clicks++,
      })
    )
  );
  const button = env.container.querySelector("button");
  assert.equal(button.tabIndex, 0);
  assert.equal(button.getAttribute("aria-pressed"), "true");
  assert.match(button.textContent, /0%/);
  assert.doesNotMatch(button.textContent, /Infinity|NaN/);
  await act(async () => button.click());
  assert.equal(clicks, 1);
  assert.equal(submit.mock.callCount(), 0);
  await env.render(React.createElement(SingleProgressStats, { title: "Completed", completed: 1, total: 2 }));
  assert.equal(env.container.querySelector("button"), null);
  assert.match(env.container.textContent, /50%/);
});

void test("quick-add and empty-state actions are native non-submit controls", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const actions = [];
  await env.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(KanbanQuickAddIssueButton, { onClick: () => actions.push("add") }),
      React.createElement(
        "ul",
        null,
        React.createElement(EmptySpaceItem, {
          title: "Create project",
          Icon,
          action: () => actions.push("create"),
        })
      )
    )
  );
  const buttons = [...env.container.querySelectorAll("button")];
  assert.equal(buttons.length, 2);
  for (const button of buttons) {
    assert.equal(button.type, "button");
    assert.equal(button.tabIndex, 0);
  }
  await act(async () => buttons.forEach((button) => button.click()));
  assert.deepEqual(actions, ["add", "create"]);
  assert.equal(env.container.querySelector("li").getAttribute("role"), null);
});

void test("timeline view controls expose the current selection and emit the selected view", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const views = [];
  await env.render(
    React.createElement(GanttChartHeader, {
      blockIds: [],
      currentView: "month",
      fullScreenMode: false,
      loaderTitle: "Items",
      showToday: false,
      handleChartView: (view) => views.push(view),
      handleToday() {},
      toggleFullScreenMode() {},
    })
  );
  const controls = [...env.container.querySelectorAll("button[aria-pressed]")];
  assert.equal(controls.length, 3);
  assert.equal(controls.filter((button) => button.getAttribute("aria-pressed") === "true").length, 1);
  await act(async () => controls.forEach((button) => button.click()));
  assert.deepEqual(views, ["week", "month", "quarter"]);
});
