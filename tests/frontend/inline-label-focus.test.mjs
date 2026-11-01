import assert from "node:assert/strict";
import test from "node:test";
import { React, act, settleQuery, fixture, button, click } from "./helpers/labels-native-ui-fixture.mjs";

await test("an explicitly opened inline label form owns initial and return focus without trapping page navigation", async (t) => {
  const view = await fixture(t, []);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  const trigger = button(view.environment.container, "common.add_label");
  await act(async () => trigger.focus());
  await click(trigger);
  await settleQuery();
  const input = view.environment.container.querySelector('input[name="name"]');
  assert.ok(document.activeElement === input, "the name field receives initial focus");
  const outside = document.createElement("button");
  document.body.append(outside);
  await act(async () => outside.focus());
  assert.ok(document.activeElement === outside, "inline editing allows focus to leave the form");
  await act(async () => input.focus());
  await click(button(view.environment.container, "cancel"));
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]'), null);
  assert.ok(document.activeElement === trigger, "closing returns focus to the opening control");
});
