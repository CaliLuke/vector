import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { PropertyActivityMessage } = await import("../../apps/web/core/components/core/property-activity-message.tsx");
const { propertyActivityDetails } = await import("../../apps/web/core/components/core/activity-property.tsx");

await test("profile activity registers and renders every custom-property event kind", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  assert.deepEqual(Object.keys(propertyActivityDetails), ["type", "custom_property", "custom_properties"]);
  await env.render(
    propertyActivityDetails.custom_property.message(
      { field: "custom_property", verb: "restored", comment: "Notes", old_value: "Later", new_value: "Earlier" },
      false
    )
  );
  assert.equal(document.body.textContent, "restored Notes from Later to Earlier.");
  assert.equal(document.querySelector("a"), null);
});

await test("property activity retains historical labels, zero, false and clear values", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(
    React.createElement(PropertyActivityMessage, {
      activity: {
        field: "custom_property",
        verb: "updated",
        comment: "Previous field name",
        old_value: "0",
        new_value: "false",
      },
    })
  );
  assert.equal(document.body.textContent, "changed Previous field name from 0 to false.");
  await env.render(
    React.createElement(PropertyActivityMessage, {
      activity: {
        field: "custom_property",
        verb: "updated",
        comment: "Notes",
        old_value: "First\nSecond",
        new_value: "",
      },
    })
  );
  assert.equal(document.body.textContent, "changed Notes from First\nSecond to empty.");
});

await test("restore and type events are distinguished and historical text is never HTML", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(
    React.createElement(PropertyActivityMessage, {
      activity: {
        field: "custom_property",
        verb: "restored",
        comment: "<b>Old label</b>",
        old_value: "Later",
        new_value: '<img src="bad" onerror="bad()">',
      },
    })
  );
  assert.equal(document.body.textContent, 'restored <b>Old label</b> from Later to <img src="bad" onerror="bad()">.');
  assert.equal(document.querySelector("img,b"), null);
  await env.render(
    React.createElement(PropertyActivityMessage, {
      activity: { field: "type", verb: "updated", old_value: "Task", new_value: "Bug" },
    })
  );
  assert.equal(document.body.textContent, "changed type from Task to Bug.");
});

await test("preservation and permanent-removal notices retain their distinct audit messages", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(
    React.createElement(PropertyActivityMessage, {
      activity: {
        field: "custom_properties",
        comment: "Removed Notes",
        old_value: "Saved value",
        new_value: "Values preserved in description.",
      },
    })
  );
  assert.equal(
    document.body.textContent,
    "Removed Notes. Values preserved in description. Previous values: Saved value"
  );
  await env.render(
    React.createElement(PropertyActivityMessage, {
      activity: {
        field: "custom_properties",
        comment: "Removed Notes",
        old_value: "",
        new_value: "Current values removed without preservation. Existing history retained.",
      },
    })
  );
  assert.equal(
    document.body.textContent,
    "Removed Notes. Current values removed without preservation. Existing history retained."
  );
});
