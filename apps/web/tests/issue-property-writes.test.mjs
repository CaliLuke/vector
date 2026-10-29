import assert from "node:assert/strict";
import test from "node:test";
import { onlineManager } from "@tanstack/react-query";
import { React, act } from "../../../tests/frontend/helpers/react.mjs";
import { createQueryRouterFixture, settleQuery } from "../../../tests/frontend/helpers/query-router.mjs";
import { registerWebTestModules } from "../../../tests/frontend/helpers/web-test-modules.mjs";

globalThis.React = React;
globalThis.propertyChoices = {};
globalThis.propertyControls = {};
globalThis.propertyToasts = [];
globalThis.propertyActions = {};
globalThis.renderPropertyControl = (name, props) => {
  globalThis.propertyControls[name] = props;
  return React.createElement(
    "button",
    {
      type: "button",
      disabled: props.disabled,
      "data-property": name,
      "data-value": JSON.stringify(props.value),
      onClick: () => (props.onChange ?? props.onSelect)(globalThis.propertyChoices[name]),
    },
    name
  );
};
const control = (name, exported) =>
  `export const ${exported} = props => globalThis.renderPropertyControl(${JSON.stringify(name)}, props);`;
registerWebTestModules({
  "@plane/i18n": "export const useTranslation = () => ({t:key=>key});",
  "@plane/propel/icons":
    "export const LinkIcon=()=>null;export const StartDatePropertyIcon=()=>null;export const ViewsIcon=()=>null;export const DueDatePropertyIcon=()=>null;",
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error'};export const setToast=value=>globalThis.propertyToasts.push(value);",
  "lucide-react": "export const Paperclip=()=>null;",
  "@/components/dropdowns/cycle": control("cycle", "CycleDropdown"),
  "@/components/dropdowns/date": control("date", "DateDropdown"),
  "@/components/dropdowns/date-range": control("range", "DateRangeDropdown"),
  "@/components/dropdowns/estimate": control("estimate", "EstimateDropdown"),
  "@/components/dropdowns/member/dropdown": control("assignee", "MemberDropdown"),
  "@/components/dropdowns/module/dropdown": control("modules", "ModuleDropdown"),
  "@/components/dropdowns/priority": control("priority", "PriorityDropdown"),
  "@/components/dropdowns/state/dropdown": control("state", "StateDropdown"),
  "./labels": control("labels", "IssuePropertyLabels"),
  "@/hooks/use-estimates": "export const useProjectEstimates=()=>({isEnabled:true});",
  "@/hooks/use-labels": "export const useProjectLabels=()=>({labelsById:new Map()});",
  "@/hooks/use-projects":
    "export const useWorkspaceProjectDetails=()=>({projects:[{id:'p1',identifier:'P',module_view:true,cycle_view:true}]});",
  "@/hooks/use-project-states": "export const useProjectStates=()=>({getStateById:()=>({group:'started'})});",
  "@/hooks/use-issue-list-type": "export const useIssueListType=()=> 'PROJECT';",
  "@/hooks/use-issues-actions": "export const useIssuesActions=()=>globalThis.propertyActions;",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
});
const { IssueProperties } = await import("../core/components/issues/issue-layouts/properties/all-properties.tsx");
const identity = await import("../core/queries/identity.ts");

const row = {
  id: "first",
  project_id: "p1",
  state_id: "started",
  priority: "low",
  assignee_ids: [],
  label_ids: [],
  module_ids: ["old", "kept"],
  cycle_id: "old-cycle",
  start_date: "2026-10-10",
  target_date: "2026-10-15",
  sequence_id: 1,
};
const displayProperties = {
  state: true,
  priority: true,
  assignee: true,
  labels: true,
  modules: true,
  cycle: true,
  estimate: true,
  start_date: true,
  due_date: true,
};
function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
async function setup(t, save = async () => {}) {
  const fixture = await createQueryRouterFixture(t);
  fixture.client.setQueryData(identity.currentUserOptions().queryKey, { id: "alice" });
  globalThis.propertyToasts = [];
  globalThis.propertyControls = {};
  globalThis.propertyChoices = {
    priority: "urgent",
    range: { from: new Date(2026, 9, 20), to: new Date(2026, 9, 25) },
    modules: ["kept", "new"],
  };
  const writes = [];
  const updateIssue = (...args) => {
    writes.push(args);
    return save(...args);
  };
  globalThis.propertyActions = {
    changeModulesInIssue: (...args) => {
      writes.push(args);
      return save(...args);
    },
    addCycleToIssue: (...args) => {
      writes.push(args);
      return save(...args);
    },
    removeCycleFromIssue: (...args) => {
      writes.push(args);
      return save(...args);
    },
  };
  const render = async (issue = row, extra = {}) =>
    fixture.render(
      React.createElement(IssueProperties, {
        issue,
        updateIssue,
        displayProperties,
        isReadOnly: false,
        className: "",
        activeLayout: "list",
        ...extra,
      })
    );
  await render();
  const button = (name) => fixture.environment.container.querySelector(`[data-property="${name}"]`);
  const click = async (name) => {
    assert.ok(button(name), name);
    await act(() => button(name).click());
    await settleQuery();
  };
  return { ...fixture, writes, render, button, click };
}

await test("date range selection writes both endpoints in one patch", async (t) => {
  const view = await setup(t);
  await view.click("range");
  assert.deepEqual(view.writes, [["p1", "first", { start_date: "2026-10-20", target_date: "2026-10-25" }]]);
});
await test("clearing a date range clears both endpoints in one patch", async (t) => {
  const view = await setup(t);
  globalThis.propertyChoices.range = undefined;
  await view.click("range");
  assert.deepEqual(view.writes, [["p1", "first", { start_date: null, target_date: null }]]);
});
await test("module changes send additions and removals together", async (t) => {
  const view = await setup(t);
  await view.click("modules");
  assert.deepEqual(view.writes, [["p1", "first", ["new"], ["old"]]]);
});
await test("an unchanged module selection performs no write", async (t) => {
  const view = await setup(t);
  globalThis.propertyChoices.modules = ["old", "kept"];
  await view.click("modules");
  assert.deepEqual(view.writes, []);
});
await test("failed property saves keep the value and allow a single retry", async (t) => {
  const first = deferred();
  const second = deferred();
  let attempt = 0;
  const view = await setup(t, () => (++attempt === 1 ? first.promise : second.promise));
  await act(() => {
    globalThis.propertyControls.priority.onChange("urgent");
    globalThis.propertyControls.priority.onChange("high");
  });
  await settleQuery();
  assert.equal(view.writes.length, 1);
  assert.equal(view.button("priority").disabled, true);
  assert.equal(view.button("state").disabled, true);
  await act(() => first.reject(new Error("rejected")));
  await settleQuery();
  assert.equal(globalThis.propertyToasts.length, 1);
  assert.equal(globalThis.propertyToasts[0].type, "error");
  assert.equal(view.button("priority").getAttribute("data-value"), '"low"');
  assert.equal(view.button("priority").disabled, false);
  await view.click("priority");
  assert.equal(view.writes.length, 2);
  await act(() => second.resolve());
  await settleQuery();
  assert.equal(view.button("priority").disabled, false);
});
await test("a late failure cannot notify or unlock a replacement row", async (t) => {
  const first = deferred();
  const second = deferred();
  let attempt = 0;
  const view = await setup(t, () => (++attempt === 1 ? first.promise : second.promise));
  await view.click("priority");
  await view.render({ ...row, id: "second" });
  assert.equal(view.button("priority").disabled, false);
  await view.click("priority");
  await act(() => first.reject(new Error("old request")));
  await settleQuery();
  assert.equal(globalThis.propertyToasts.length, 0);
  assert.equal(view.button("priority").disabled, true);
  await act(() => second.resolve());
  await settleQuery();
  assert.deepEqual(
    view.writes.map((write) => write[1]),
    ["first", "second"]
  );
});
await test("a renewed session does not inherit a pending save or its error", async (t) => {
  const first = deferred();
  const view = await setup(t, () => first.promise);
  await view.click("priority");
  await act(() => identity.advanceIdentitySession(view.client));
  await settleQuery();
  assert.equal(view.button("priority").disabled, false);
  await act(() => first.reject(new Error("previous session")));
  await settleQuery();
  assert.equal(globalThis.propertyToasts.length, 0);
});
await test("read-only property controls do not write", async (t) => {
  const view = await setup(t);
  await view.render(row, { isReadOnly: true });
  for (const name of ["priority", "state", "range", "modules", "cycle", "estimate", "labels"]) {
    assert.equal(view.button(name).disabled, true, name);
  }
  await act(() => {
    for (const button of view.environment.container.querySelectorAll("button")) button.click();
  });
  assert.deepEqual(view.writes, []);
});
await test("scalar and multi-value properties preserve their patch payloads", async (t) => {
  const view = await setup(t);
  globalThis.propertyChoices = {
    state: "done",
    priority: "urgent",
    assignee: ["member"],
    labels: ["label"],
    estimate: "point",
  };
  await view.click("state");
  await view.click("priority");
  await view.click("assignee");
  await view.click("labels");
  await view.click("estimate");
  assert.deepEqual(view.writes, [
    ["p1", "first", { state_id: "done" }],
    ["p1", "first", { priority: "urgent" }],
    ["p1", "first", { assignee_ids: ["member"] }],
    ["p1", "first", { label_ids: ["label"] }],
    ["p1", "first", { estimate_point: "point" }],
  ]);
});
await test("cycle selection preserves add, removal, and unchanged behavior", async (t) => {
  const view = await setup(t);
  globalThis.propertyChoices.cycle = "old-cycle";
  await view.click("cycle");
  assert.deepEqual(view.writes, []);
  globalThis.propertyChoices.cycle = "new-cycle";
  await view.click("cycle");
  globalThis.propertyChoices.cycle = null;
  await view.click("cycle");
  assert.deepEqual(view.writes, [
    ["p1", "new-cycle", "first"],
    ["p1", "old-cycle", "first"],
  ]);
});
await test("callbacks from a replaced row cannot start another write", async (t) => {
  const view = await setup(t);
  const oldChange = globalThis.propertyControls.priority.onChange;
  await view.render({ ...row, id: "second" });
  await act(() => oldChange("urgent"));
  await settleQuery();
  assert.deepEqual(view.writes, []);
  await view.click("priority");
  assert.deepEqual(view.writes, [["p1", "second", { priority: "urgent" }]]);
});
await test("an unmounted row does not report a late failure", async (t) => {
  const pending = deferred();
  const view = await setup(t, () => pending.promise);
  await view.click("priority");
  await view.environment.render(null);
  await act(() => pending.reject(new Error("late request")));
  await settleQuery();
  assert.deepEqual(globalThis.propertyToasts, []);
});
await test("missing patch support disables patch fields while assignments remain available", async (t) => {
  const view = await setup(t);
  await view.render(row, { updateIssue: undefined });
  assert.equal(view.button("priority").disabled, true);
  assert.equal(view.button("range").disabled, true);
  assert.equal(view.button("modules").disabled, false);
  await view.click("modules");
  assert.deepEqual(view.writes, [["p1", "first", ["new"], ["old"]]]);
});

await test("a queued offline save cannot run after its row is replaced", async (t) => {
  const view = await setup(t);
  t.after(() => onlineManager.setOnline(true));
  onlineManager.setOnline(false);
  await view.click("priority");
  assert.deepEqual(view.writes, []);
  assert.equal(view.button("priority").disabled, true);
  await view.render({ ...row, id: "second" });
  await act(() => onlineManager.setOnline(true));
  await settleQuery();
  assert.deepEqual(view.writes, []);
  assert.deepEqual(globalThis.propertyToasts, []);
  await view.click("priority");
  assert.deepEqual(view.writes, [["p1", "second", { priority: "urgent" }]]);
});
