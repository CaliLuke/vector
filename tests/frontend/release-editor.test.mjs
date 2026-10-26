import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
globalThis.releaseReact = React;
const stubs = {
  "@plane/propel/button": `export const Button=({loading,variant,...props})=>globalThis.releaseReact.createElement("button",props);`,
  "@/components/work-item-types/controls": `const R=globalThis.releaseReact;export const inputStyle="";export const Field=({label,children})=>R.createElement("label",null,label,children(label));export const PreviewDialog=({title,children,footer})=>R.createElement("section",null,R.createElement("h2",null,title),children,footer);`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const bootstrap = await createReactDomTestEnvironment();
const { ReleaseEditor } = await import("../../apps/web/core/components/releases/release-editor.tsx");
const { ReleaseRemoval } = await import("../../apps/web/core/components/releases/release-removal.tsx");
await bootstrap.unmount();
const saved = {
  id: "release",
  name: "October",
  description: "Saved description",
  target_date: "2026-10-31",
  created_at: "2026-10-01",
};
async function fixture(t, Component = ReleaseEditor, release = saved) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const f = {
    env,
    commands: [],
    closed: 0,
    failure: undefined,
    catalog: {
      revision: 8,
      releases: [{ ...saved, name: "Remote name", description: "Remote description", target_date: "2026-11-01" }],
    },
  };
  f.props = {
    release: release ?? undefined,
    revision: 7,
    pending: false,
    onSave: async (input) => {
      f.commands.push(input);
      if (f.failure) throw f.failure;
    },
    onReload: async () => {
      if (f.reloadFailure) throw f.reloadFailure;
      return f.catalog;
    },
    onClose: () => f.closed++,
  };
  f.render = () => env.render(React.createElement(Component, f.props));
  f.click = (name) =>
    act(async () => [...env.container.querySelectorAll("button")].find((b) => b.textContent === name).click());
  f.submit = () =>
    act(async () =>
      env.container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    );
  f.value = (id) => env.container.querySelector(`[id="${id}"]`).value;
  f.change = (id, value) =>
    act(async () => {
      const input = env.container.querySelector(`[id="${id}"]`);
      const prototype =
        input instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
      Object.getOwnPropertyDescriptor(prototype, "value").set.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  await f.render();
  return f;
}
await test("failed release saves retain text and retry the same creation identity", async (t) => {
  const f = await fixture(t, ReleaseEditor, null);
  await f.change("Name", "Release one");
  f.failure = { status: 500 };
  await f.submit();
  assert.equal(f.closed, 0);
  assert.equal(f.value("Name"), "Release one");
  f.failure = undefined;
  await f.submit();
  assert.equal(f.commands.length, 2);
  assert.equal(f.commands[0].id, f.commands[1].id);
  assert.equal(f.commands[1].values.name, "Release one");
  assert.equal(f.closed, 1);
});
await test("catalog conflict requires reload and preserves edited fields while refreshing untouched fields", async (t) => {
  const f = await fixture(t);
  await f.change("Name", "Local name");
  f.failure = { status: 409 };
  await f.submit();
  await f.submit();
  assert.equal(f.commands.length, 1);
  await f.click("Reload current values");
  assert.equal(f.value("Name"), "Local name");
  assert.equal(f.value("Description"), "Remote description");
  assert.equal(f.value("Target date"), "2026-11-01");
  f.failure = undefined;
  await f.submit();
  assert.equal(f.commands[1].revision, 8);
  assert.equal(f.commands[1].values.name, "Local name");
  assert.deepEqual(Object.keys(f.commands[1].values).toSorted(), ["description", "name", "target_date"]);
});
await test("reload recognizes an accepted create and reviews further edits as an update", async (t) => {
  const f = await fixture(t, ReleaseEditor, null);
  await f.change("Name", "Release one");
  f.failure = { status: 500 };
  await f.submit();
  const id = f.commands[0].id;
  await f.change("Description", "Local notes");
  f.failure = { status: 409 };
  await f.submit();
  f.catalog = { revision: 9, releases: [{ ...saved, id, name: "Release one", description: "", target_date: "" }] };
  await f.click("Reload current values");
  assert.equal(f.value("Description"), "Local notes");
  assert.match(f.env.container.textContent, /Edit release/);
  f.failure = undefined;
  await f.submit();
  assert.equal(f.commands[2].action, "update");
  assert.equal(f.commands[2].id, id);
  assert.equal(f.commands[2].revision, 9);
});
await test("deleted release retains the local draft without offering a save", async (t) => {
  const f = await fixture(t);
  await f.change("Description", "Keep this text");
  f.failure = { status: 404 };
  await f.submit();
  f.catalog = { revision: 9, releases: [] };
  await f.click("Reload current values");
  assert.equal(f.value("Description"), "Keep this text");
  assert.match(f.env.container.textContent, /was removed/);
  await f.submit();
  assert.equal(f.commands.length, 1);
  assert.equal(f.closed, 0);
});
await test("failed conflict reload retains draft and can be retried", async (t) => {
  const f = await fixture(t);
  await f.change("Name", "Local");
  f.failure = { status: 409 };
  await f.submit();
  f.reloadFailure = new Error("Offline");
  await f.click("Reload current values");
  assert.equal(f.value("Name"), "Local");
  assert.equal(f.closed, 0);
  assert.match(f.env.container.textContent, /Offline/);
  f.reloadFailure = undefined;
  await f.click("Reload current values");
  assert.equal(f.value("Name"), "Local");
});
await test("delete reload requires a new confirmation of the current release", async (t) => {
  const f = await fixture(t, ReleaseRemoval);
  f.failure = { status: 409 };
  await f.click("Delete release");
  await f.click("Reload release");
  assert.match(f.env.container.textContent, /Delete Remote name/);
  assert.equal(f.commands.length, 1);
  f.failure = undefined;
  await f.click("Delete release");
  assert.equal(f.commands[1].revision, 8);
  assert.equal(f.closed, 1);
});
await test("remote deletion turns confirmation into a close action", async (t) => {
  const f = await fixture(t, ReleaseRemoval);
  f.failure = { status: 404 };
  await f.click("Delete release");
  f.catalog = { revision: 8, releases: [] };
  await f.click("Reload release");
  assert.match(f.env.container.textContent, /no longer/);
  assert.equal(f.commands.length, 1);
  await f.click("Close");
  assert.equal(f.closed, 1);
});
