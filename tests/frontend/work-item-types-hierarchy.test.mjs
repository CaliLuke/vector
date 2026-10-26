import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";

const stubs = {
  "@plane/propel/button": `const React=globalThis.hierarchyReact; export const Button=({variant,prependIcon,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
  "@tanstack/react-router": `const React=globalThis.hierarchyReact; export const useParams=()=>({workspaceSlug:"test-workspace"}); export const Link=({to,params,search,children,...props})=>React.createElement("a",{...props,href:to.replace("$workspaceSlug",params.workspaceSlug)},children);`,
  "./controls": `const React=globalThis.hierarchyReact;
    export const TypeIcon=()=>null;
    export const EmptyState=({title,children})=>React.createElement("section",null,title,children);
    export const Confirmation=({title,action,children,onConfirm,onClose})=>React.createElement("section",{role:"dialog","aria-label":title},children,React.createElement("button",{onClick:onConfirm},action),React.createElement("button",{onClick:onClose},"Cancel"));`,
};
const nativeForms = registerWorkItemTypesFormHarness({
  sourceFiles: ["apps/web/core/components/work-item-types/hierarchy.tsx"],
  stubs,
  reactGlobal: "hierarchyReact",
  react: React,
});
const { HierarchyPreview } = await import("../../apps/web/core/components/work-item-types/hierarchy.tsx");
const { WorkItemTypesPreviewProvider, useTypePreview } =
  await import("../../apps/web/core/components/work-item-types/preview-provider.tsx");

function Screen({ readOnly }) {
  const { content } = useTypePreview();
  const [revision, setRevision] = React.useState(0);
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(HierarchyPreview, { key: revision, readOnly }),
    React.createElement("button", { onClick: () => setRevision((value) => value + 1) }, "Reopen hierarchy"),
    React.createElement("output", { "data-testid": "saved" }, JSON.stringify(content))
  );
}
const click = (node) => act(async () => node.click());
const drop = (target, id) =>
  act(async () => {
    const event = new Event("drop", { bubbles: true, cancelable: true });
    Object.defineProperty(event, "dataTransfer", { value: { getData: () => id } });
    target.dispatchEvent(event);
  });
async function setup(t, readOnly = false) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(Screen, { readOnly })));
  const button = (name, scope = env.container) =>
    [...scope.querySelectorAll("button")].find((node) => node.textContent === name);
  const level = (index) => env.container.querySelector(`section[aria-label="Level ${index}"]`);
  const saved = () => JSON.parse(env.container.querySelector('[data-testid="saved"]').textContent);
  const move = (name, value) =>
    changeFormControl(env.container.querySelector(`select[aria-label="Move ${name} to level"]`), String(value));
  return { env, button, level, saved, move };
}

await test("default-only edits can be discarded and saved across reopening", async (t) => {
  const { button, level, saved } = await setup(t);
  await click(button("Set as default", level(2)));
  assert.equal(button("Save changes").disabled, false);
  assert.equal(saved().hierarchyDefaultLevel, 0);
  await click(button("Discard"));
  assert.ok(button("Set as default", level(2)));
  assert.equal(button("Save changes").disabled, true);
  await click(button("Set as default", level(2)));
  await click(button("Save changes"));
  assert.equal(saved().hierarchyDefaultLevel, 2);
  assert.equal(
    nativeForms.getForms("apps/web/core/components/work-item-types/hierarchy.tsx").at(-1).state.isSubmitSuccessful,
    true
  );
  assert.equal(button("Save changes").disabled, true);
  await click(button("Reopen hierarchy"));
  assert.equal(button("Set as default", level(2)), undefined);
});

await test("moving types creates levels, removes empty ones, and discards the whole draft", async (t) => {
  const { env, button, level, saved, move } = await setup(t);
  const original = saved();
  assert.equal(env.container.querySelector("a").getAttribute("href"), "/test-workspace/settings/work-item-types");
  await move("Bug", 3);
  assert.ok(level(3));
  assert.equal(button("Save changes").disabled, false);
  await click(button("Set as default", level(3)));
  await move("Bug", 1);
  assert.equal(level(3), null);
  assert.equal(button("Set as default", level(0)), undefined);
  assert.equal(button("Save changes").disabled, true);
  await move("Epic", 1);
  assert.equal(level(2), null);
  await click(button("Discard"));
  assert.ok(level(2));
  assert.equal(level(2).querySelector("select").value, "2");
  assert.deepEqual(saved(), original);
});

await test("dragging compacts intermediate levels and remaps the default without losing types", async (t) => {
  const { env, button, level, saved, move } = await setup(t);
  await click(button("Set as default", level(2)));
  await drop(env.container.querySelector('[aria-label="New hierarchy level"]'), "incident");
  assert.ok(level(3));
  await move("Story", 0);
  await move("Bug", 0);
  assert.equal(level(3), null);
  assert.equal(level(1).querySelector('select[aria-label="Move Epic to level"]').value, "1");
  assert.equal(button("Set as default", level(1)), undefined);
  assert.equal(env.container.querySelectorAll("select").length, saved().types.length);
  await drop(level(0), "unknown-type");
  assert.equal(env.container.querySelectorAll("select").length, saved().types.length);
});

await test("conflict review preserves the draft on cancel and removes only invalid links on save", async (t) => {
  const { env, button, saved, move } = await setup(t);
  const original = saved();
  await move("Task", 1);
  await click(button("Save changes"));
  assert.ok(env.container.querySelector('[role="dialog"]'));
  assert.deepEqual(saved(), original);
  await click(button("Cancel"));
  assert.equal(button("Save changes").disabled, false);
  await click(button("Save changes"));
  await click(button("Save anyway"));
  const changed = saved();
  assert.equal(changed.items.length, original.items.length);
  assert.equal(changed.items.find((item) => item.id === "WEB-143").parent, "");
  assert.equal(changed.items.find((item) => item.id === "WEB-142").parent, "WEB-100");
  assert.equal(button("Save changes").disabled, true);
});

await test("read-only hierarchy ignores drops and disables editing controls", async (t) => {
  const { env, button, saved, level } = await setup(t, true);
  const original = saved();
  await drop(env.container.querySelector('[aria-label="New hierarchy level"]'), "bug");
  assert.equal(level(3), null);
  assert.equal(button("Save changes").disabled, true);
  assert.equal(button("Set as default", level(2)).disabled, true);
  assert.ok([...env.container.querySelectorAll("select")].every((select) => select.disabled));
  assert.deepEqual(saved(), original);
});
