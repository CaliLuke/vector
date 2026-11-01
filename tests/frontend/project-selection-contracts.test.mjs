import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { setPositioningGeometry } from "./helpers/positioning.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "./controls": "export const TypeIcon=()=>null;",
});
const bootstrap = await createReactDomTestEnvironment();
const { ModuleStatusSelect } = await import("../../apps/web/core/components/modules/select/status.tsx");
const { ProjectTypeDefault } = await import("../../apps/web/core/components/work-item-types/project-type-default.tsx");
await bootstrap.unmount();

async function choose(env, label) {
  const trigger = env.container.querySelector("button");
  setPositioningGeometry(trigger);
  await act(async () => trigger.click());
  const option = [...document.querySelectorAll('[role="option"]')].find((item) => item.textContent.trim() === label);
  assert.ok(option, `choice ${label} is available`);
  await act(async () => {
    option.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    option.click();
  });
}

void test("module status selection updates the controlled value and label", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const changes = [];
  function Harness() {
    const [value, setValue] = React.useState("backlog");
    return React.createElement(ModuleStatusSelect, {
      value,
      onChange: (next) => {
        changes.push(next);
        setValue(next);
      },
    });
  }
  await env.render(React.createElement(Harness));
  await choose(env, "project_modules.status.completed");
  assert.deepEqual(changes, ["completed"]);
  assert.match(env.container.querySelector("button").textContent, /completed/);
});

void test("empty module status remains selectable while a read-only badge blocks changes", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const changes = [];
  const onChange = (value) => changes.push(value);
  await env.render(React.createElement(ModuleStatusSelect, { value: null, onChange, error: true }));
  assert.match(env.container.querySelector("button").textContent, /Status/);
  await choose(env, "project_modules.status.backlog");
  assert.deepEqual(changes, ["backlog"]);
  await env.render(
    React.createElement(ModuleStatusSelect, { value: "completed", onChange, disabled: true, badge: true })
  );
  const trigger = env.container.querySelector("button");
  assert.equal(trigger.disabled, true);
  assert.match(trigger.textContent, /completed/);
  await act(async () => trigger.click());
  assert.deepEqual(changes, ["backlog"]);
});

void test("project defaults retain the saved unavailable type and allow an available type or Automatic", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const types = [
    { id: "task", name: "Task", available: true },
    { id: "saved", name: "Saved retired type", available: false },
    { id: "hidden", name: "Other retired type", available: false },
  ];
  const changes = [];
  function Harness() {
    const [value, setValue] = React.useState("saved");
    return React.createElement(ProjectTypeDefault, {
      types,
      value,
      disabled: false,
      onChange: (next) => {
        changes.push(next);
        setValue(next);
      },
    });
  }
  await env.render(React.createElement(Harness));
  assert.match(env.container.querySelector("button").textContent, /Saved retired typeUnavailable/);
  const trigger = env.container.querySelector("button");
  setPositioningGeometry(trigger);
  await act(async () => trigger.click());
  const options = [...document.querySelectorAll('[role="option"]')];
  const unavailable = options.find((item) => item.textContent.trim() === "Saved retired type");
  assert.ok(unavailable, `options: ${JSON.stringify(options.map((item) => item.textContent))}`);
  assert.equal(unavailable.getAttribute("aria-disabled"), "true");
  assert.equal(
    options.some((item) => item.textContent.includes("Other retired type")),
    false
  );
  await act(async () => unavailable.click());
  assert.deepEqual(changes, []);
  const task = options.find((item) => item.textContent.trim() === "Task");
  await act(async () => {
    task.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    task.click();
  });
  assert.deepEqual(changes, ["task"]);
  assert.equal(env.container.querySelector("button").textContent, "Task");
  await choose(env, "Automatic");
  assert.deepEqual(changes, ["task", ""]);
  assert.equal(env.container.querySelector("button").textContent, "Automatic");
  await env.render(
    React.createElement(ProjectTypeDefault, {
      types,
      value: "task",
      disabled: true,
      onChange: (value) => changes.push(value),
    })
  );
  assert.equal(env.container.querySelector("button").disabled, true);
  await act(async () => env.container.querySelector("button").click());
  assert.deepEqual(changes, ["task", ""]);
});
