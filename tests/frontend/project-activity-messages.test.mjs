import assert from "node:assert/strict";
import test from "node:test";
import { React, webRequire } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { messages } = await import("../../apps/web/core/components/common/activity/helper.tsx");
const { getCurrentHookAsCSV } = await import("../../apps/web/core/components/web-hooks/utils.ts");
const { renderToStaticMarkup } = webRequire("react-dom/server");

void test("project history retains toggle semantics, defaults and archive attribution", () => {
  for (const [field, newValue, oldValue, expected] of [
    ["module_view", "false", undefined, "disabled module view"],
    ["issue_views_view", "true", undefined, "enabled issue views view"],
    ["page_view", undefined, undefined, "changed page view"],
    ["is_epic_enabled", "true", undefined, "enabled epics"],
    ["is_time_tracking_enabled", "false", undefined, "disabled time tracking"],
    ["description", "", undefined, "removed the project description"],
    ["start_date", undefined, undefined, "removed the start date"],
    ["network", "false", undefined, "enabled network access"],
    ["new_field", undefined, undefined, "changed new field "],
    ["constructor", undefined, undefined, "changed constructor "],
  ]) {
    const { message } = messages({ field, new_value: newValue, old_value: oldValue, verb: "changed" }, undefined);
    assert.equal(renderToStaticMarkup(React.createElement(React.Fragment, null, message)), expected);
  }
  assert.deepEqual(messages({ field: "archived_at", new_value: "archive" }), {
    message: "archived the project",
    customUserName: "Vector",
  });
  assert.deepEqual(messages({ field: "archived_at", new_value: "restore" }), {
    message: "restored the project",
    customUserName: undefined,
  });
});

void test("project history preserves scoped cycle links, removed labels and escaped names", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const activity = {
    field: "cycles",
    verb: "added",
    project: "project",
    new_identifier: "cycle",
    new_value: "<b>Cycle</b>",
  };
  await env.render(messages(activity, { slug: "workspace" }).message);
  const link = env.container.querySelector("a");
  assert.equal(link.getAttribute("href"), "/workspace/projects/project/cycles/cycle");
  assert.equal(link.textContent, "<b>Cycle</b>");
  assert.equal(link.querySelector("b"), null);
  assert.equal(link.rel, "noopener noreferrer");
  await env.render(
    messages({ ...activity, verb: "removed", old_value: "Retired cycle" }, { slug: "workspace" }).message
  );
  assert.equal(env.container.querySelector("a"), null);
  assert.equal(env.container.textContent, "removed this project from the cycle Retired cycle");
  await env.render(messages({ field: "priority", new_value: "High <priority>" }).message);
  assert.equal(env.container.textContent, "set the priority to High <priority>");
  assert.equal(env.container.querySelector("priority"), null);
  await env.render(messages({ field: "state", new_value: "" }).message);
  assert.equal(env.container.textContent, "set the state to none");
  await env.render(messages({ field: "estimate", old_value: "5", verb: "updated" }).message);
  assert.equal(env.container.textContent, "removed the estimate point 5");
});

void test("webhook CSV preserves false subscriptions and uses empty fields for unavailable data", () => {
  const empty = getCurrentHookAsCSV(null, undefined, undefined);
  assert.ok(Object.values(empty).every((value) => value === ""));
  const csv = getCurrentHookAsCSV(
    { name: "Workspace" },
    {
      id: "hook",
      url: "https://example.com",
      is_active: false,
      project: true,
      issue: false,
      module: true,
      cycle: false,
      issue_comment: true,
    },
    "secret"
  );
  assert.equal(csv.is_active, "false");
  assert.equal(csv.project, "true");
  assert.equal(csv.issue, "false");
  assert.equal(csv.workspace, "Workspace");
  assert.equal(csv.secret_key, "secret");
});
