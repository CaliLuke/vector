import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();

for (const [name, file, exportName] of [
  ["project", "project", "ProjectOrderByDropdown"],
  ["module", "modules", "ModuleOrderByDropdown"],
]) {
  void test(`${name} ordering uses valid direction values and never reverses manual ordering`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const { [exportName]: Dropdown } = await import(`../../apps/web/core/components/${file}/dropdowns/order-by.tsx`);
    const changes = [];
    await env.render(React.createElement(Dropdown, { value: "-name", onChange: (value) => changes.push(value) }));
    await act(async () => env.container.querySelector("button").click());
    const ascending = [...document.querySelectorAll("[role='menuitem']")].find(
      (item) => item.textContent === "Ascending"
    );
    assert.ok(ascending);
    await act(async () => ascending.click());
    assert.deepEqual(changes, ["name"]);
    await act(async () => env.container.querySelector("button").click());
    const manual = [...document.querySelectorAll("[role='menuitem']")].find((item) => /manual/i.test(item.textContent));
    assert.ok(manual);
    await act(async () => manual.click());
    assert.deepEqual(changes, ["name", "sort_order"]);
    await env.render(React.createElement(Dropdown, { value: "sort_order", onChange: (value) => changes.push(value) }));
    await act(async () => env.container.querySelector("button").click());
    const directions = [...document.querySelectorAll("[role='menuitem']")].filter((item) =>
      /^(Ascending|Descending)$/.test(item.textContent)
    );
    assert.equal(
      directions.every((item) => item.disabled),
      true
    );
    await act(async () => directions.forEach((item) => item.click()));
    assert.deepEqual(changes, ["name", "sort_order"]);
  });
}

await test("applied filter enumeration preserves selection order and ignores unknown stored fields", async () => {
  const { orderedFilterKeys } = await import("../../apps/web/core/components/common/applied-filters/keys.ts");
  const filters = Object.freeze({ members: ["a"], obsolete: ["x"], created_at: ["today"], favorites: false });
  assert.deepEqual(orderedFilterKeys(filters, ["created_at", "members", "favorites"]), [
    "members",
    "created_at",
    "favorites",
  ]);
  assert.deepEqual(orderedFilterKeys({}, ["members"]), []);
});

async function chooseOrder(env, label) {
  await act(async () => env.container.querySelector("button").click());
  const item = [...document.querySelectorAll('[role="menuitem"]')].find((option) => label.test(option.textContent));
  assert.ok(item, `ordering menu includes ${label}`);
  await act(async () => item.click());
}

await test("project ordering preserves direction when changing fields and ignores the active direction", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { ProjectOrderByDropdown } = await import("../../apps/web/core/components/project/dropdowns/order-by.tsx");
  const changes = [];
  function Harness() {
    const [value, setValue] = React.useState("name");
    return React.createElement(ProjectOrderByDropdown, {
      value,
      onChange(next) {
        changes.push(next);
        setValue(next);
      },
    });
  }
  await env.render(React.createElement(Harness));
  await chooseOrder(env, /^Ascending$/);
  assert.deepEqual(changes, []);
  await chooseOrder(env, /^Descending$/);
  assert.deepEqual(changes, ["-name"]);
  await chooseOrder(env, /^Descending$/);
  assert.deepEqual(changes, ["-name"]);
  await chooseOrder(env, /created/i);
  assert.deepEqual(changes, ["-name", "-created_at"]);
  await chooseOrder(env, /^Ascending$/);
  await chooseOrder(env, /name/i);
  assert.deepEqual(changes, ["-name", "-created_at", "created_at", "name"]);
});

await test("an unset ordering disables directions and starts a chosen field in ascending order", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { ProjectOrderByDropdown } = await import("../../apps/web/core/components/project/dropdowns/order-by.tsx");
  const changes = [];
  await env.render(
    React.createElement(ProjectOrderByDropdown, { value: undefined, onChange: (value) => changes.push(value) })
  );
  await act(async () => env.container.querySelector("button").click());
  const items = [...document.querySelectorAll('[role="menuitem"]')];
  const ascending = items.find((item) => item.textContent === "Ascending");
  const descending = items.find((item) => item.textContent === "Descending");
  assert.equal(ascending.disabled, true);
  assert.equal(descending.disabled, true);
  await act(async () => {
    ascending.click();
    descending.click();
  });
  assert.deepEqual(changes, []);
  const name = items.find((item) => /name/i.test(item.textContent));
  assert.ok(name);
  await act(async () => name.click());
  assert.deepEqual(changes, ["name"]);
});
