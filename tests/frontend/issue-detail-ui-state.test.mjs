import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { IssueDetailUiStateProvider, useIssueDetailUiState } =
  await import("../../apps/web/core/providers/issue-detail-ui-state.tsx");

await test("dialog and widget setters keep stable identities while UI data changes", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  let firstToggle;
  function Probe() {
    const state = useIssueDetailUiState();
    firstToggle ??= state.toggleOpenWidget;
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("output", { id: "setter-stable" }, String(state.toggleOpenWidget === firstToggle)),
      React.createElement("output", { id: "widget-count" }, String(state.openWidgets.length)),
      React.createElement("button", { onClick: () => state.toggleOpenWidget("sub-work-items") }, "Toggle widget")
    );
  }
  await environment.render(React.createElement(IssueDetailUiStateProvider, null, React.createElement(Probe)));
  assert.equal(environment.container.querySelector("#setter-stable").textContent, "true");
  await act(async () => environment.container.querySelector("button").click());
  assert.equal(environment.container.querySelector("#setter-stable").textContent, "true");
  assert.equal(environment.container.querySelector("#widget-count").textContent, "2");
});
