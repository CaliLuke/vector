import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { setup, issueProps } from "./helpers/issue-form-harness.mjs";
import { registerHooks } from "./helpers/module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
const modalStubs = {
  "@plane/ui": `const React=globalThis.issueFormReact;export const EModalPosition={TOP:'top'},EModalWidth={XXL:'xxl'};export const ModalCore=({isOpen,children})=>isOpen?React.createElement('section',{role:'dialog'},children):null;`,
};
registerHooks({
  resolve(specifier, context, next) {
    return context.parentURL?.endsWith("/confirm-issue-discard.tsx")
      ? resolvePackageStub(specifier, modalStubs, context, next)
      : next(specifier, context);
  },
});
async function renderDraft(t, props) {
  const env = await setup(t);
  const { DraftIssueLayout } = await import("../../apps/web/core/components/issues/issue-modal/draft-issue-layout.tsx");
  await env.render(React.createElement(DraftIssueLayout, props));
  return env;
}
const button = (label) => [...document.querySelectorAll("button")].find((item) => item.textContent === label);

for (const { label, changesMade } of [
  { label: "an untouched form", changesMade: null },
  {
    label: "default-only values",
    changesMade: {
      project_id: "first",
      name: "",
      state_id: null,
      cycle_id: undefined,
      assignee_ids: [],
      custom_properties: {},
      priority: "none",
      description_html: "<p></p>",
    },
  },
]) {
  void test(`closing ${label} does not offer to save an empty draft`, async (t) => {
    let closes = 0;
    await renderDraft(t, {
      ...issueProps({ data: { name: "", description_html: "<p></p>" }, onClose: () => closes++ }),
      changesMade,
      onChange: () => {},
    });
    await act(async () => button("discard").click());
    assert.equal(closes, 1);
    assert.equal(Boolean(button("Save to Drafts")), false);
  });
}

await test("cancelling draft dismissal retains the draft, while explicit discard clears it and closes", async (t) => {
  let closes = 0,
    saves = 0;
  const changes = [];
  await renderDraft(t, {
    ...issueProps({
      data: { name: "Unsaved title", description_html: "<p></p>" },
      onClose: () => closes++,
      onSubmit: async () => saves++,
    }),
    changesMade: { name: "Unsaved title" },
    onChange: (value) => changes.push(value),
  });
  await act(async () => button("discard").click());
  assert.ok(button("Save to Drafts"));
  const changesBeforeCancel = changes.length;
  await act(async () => button("Cancel").click());
  assert.equal(Boolean(button("Save to Drafts")), false);
  assert.equal(closes, 0);
  assert.equal(changes.length, changesBeforeCancel);
  await act(async () => button("discard").click());
  await act(async () => button("Discard").click());
  assert.equal(changes.at(-1), null);
  assert.equal(changes.length, changesBeforeCancel + 1);
  assert.equal(closes, 1);
  assert.equal(saves, 0);
});

await test("an image-only description is meaningful draft content", async (t) => {
  let closes = 0;
  await renderDraft(t, {
    ...issueProps({ data: { name: "", description_html: "<p></p>" }, onClose: () => closes++ }),
    changesMade: { description_html: '<p><img src="https://example.test/diagram.png"></p>' },
    onChange: () => {},
  });
  await act(async () => button("discard").click());
  assert.ok(button("Save to Drafts"));
  assert.equal(closes, 0);
});
