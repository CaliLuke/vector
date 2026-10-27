import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { setup } from "./helpers/issue-form-harness.mjs";

const task = { id: "task", name: "Task", icon: "task", color: "#ef4444", available: true };
const bug = { ...task, id: "bug", name: "Bug", available: false };
const epic = { ...task, id: "epic", name: "Epic" };
const fields = {
  choicesQuery: { data: { enabled: true, types: [task, bug, epic] } },
  schema: { eligible_type_ids: ["task"] },
  selected: task,
  unavailable: false,
};

await test("type choices hide unavailable and hierarchy-ineligible types", async (t) => {
  const env = await setup(t);
  const { IssueTypeSelect } =
    await import("../../apps/web/core/components/issues/issue-modal/components/type-fields.tsx");
  const changes = [];
  await env.render(
    React.createElement(IssueTypeSelect, {
      fields: { ...fields, selected: undefined },
      value: null,
      onChange: (id) => changes.push(id),
      beforeChange: () => true,
    })
  );
  await act(async () => env.container.querySelector('[aria-label="Work item type"]').click());
  const options = [...document.querySelectorAll('[role="option"]')];
  assert.deepEqual(
    options.map((option) => option.textContent.trim()),
    ["Task"]
  );
  await act(async () => {
    options[0].dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true }));
    options[0].click();
  });
  assert.deepEqual(changes, ["task"]);
});

await test("retained unavailable selection remains recognizable without becoming a replacement choice", async (t) => {
  const env = await setup(t);
  const { IssueTypeSelect } =
    await import("../../apps/web/core/components/issues/issue-modal/components/type-fields.tsx");
  const changes = [];
  await env.render(
    React.createElement(IssueTypeSelect, {
      fields: { ...fields, selected: bug, unavailable: true },
      value: "bug",
      onChange: (id) => changes.push(id),
      beforeChange: () => true,
    })
  );
  const trigger = env.container.querySelector('[aria-label="Work item type"]');
  assert.equal(trigger.textContent, "Bug");
  await act(async () => trigger.click());
  assert.deepEqual(
    [...document.querySelectorAll('[role="option"]')].map((option) => option.textContent.trim()),
    ["Task"]
  );
  assert.deepEqual(changes, [], "opening must not replace a retained selection");
  assert.match(env.container.textContent, /not available for new work items/);
});
