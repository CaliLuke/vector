import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl as change, submitForm as submit } from "./helpers/native-forms.mjs";
import { setup, state, issueProps } from "./helpers/issue-form-harness.mjs";
import {
  propertyDefinition,
  enableTypeFields,
  customControl,
  chooseWorkItemType,
} from "./helpers/issue-custom-fields.mjs";

async function editSavedNotes(env) {
  const saved = [];
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        data: {
          id: "saved",
          project_id: "first",
          type_id: "task",
          name: "Existing",
          description_html: "<p></p>",
          custom_properties: { notes: "A" },
        },
        onSubmit: async (value) => saved.push(value),
      })
    )
  );
  await change(customControl(env.container, "notes"), "B");
  return saved;
}

await test("switching an existing item's type submits edited former properties for preservation", async (t) => {
  const env = await setup(t);
  enableTypeFields((input) =>
    input.type_id === "bug" ? [] : [propertyDefinition("notes", "Text", { format: "Single line" })]
  );
  const saved = await editSavedNotes(env);
  await chooseWorkItemType(env.container, "bug");
  await submit(env.container);
  assert.equal(saved.length, 1);
  assert.equal(saved[0].type_id, "bug");
  assert.equal(saved[0].custom_properties.notes, "B");
});

async function checkRecoveredParent(t, parent) {
  const env = await setup(t);
  enableTypeFields((input) =>
    input.type_id === "bug" ? [] : [propertyDefinition("notes", "Text", { format: "Single line" })]
  );
  const schema = state.schema;
  const scopes = [];
  state.schema = async (input) => {
    scopes.push(input);
    return schema(input);
  };
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const persisted = {
    id: "saved",
    project_id: "first",
    type_id: "task",
    parent_id: "old-parent",
    name: "Existing",
    description_html: "<p></p>",
    custom_properties: { notes: "A" },
  };
  scopes.length = 0;
  const saved = [];
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isDraft: true,
        persistedData: persisted,
        data: {
          ...persisted,
          project_id: "second",
          type_id: "bug",
          parent_id: parent,
          custom_properties: { notes: "B" },
        },
        onSubmit: async (value) => saved.push(value),
      })
    )
  );
  assert.ok(
    scopes.some(
      (scope) => scope.project_id === "second" && scope.type_id === "bug" && scope.parent_id === (parent ?? "")
    ),
    "destination schema retains the changed or cleared parent"
  );
  assert.ok(
    scopes.some((scope) => scope.project_id === "first" && scope.type_id === "task"),
    "former schema uses saved type and project"
  );
  await submit(env.container);
  assert.equal(saved.length, 1);
  assert.equal(saved[0].custom_properties.notes, "B");
}
await test("a recovered parent change keeps the canonical source type distinct", (t) =>
  checkRecoveredParent(t, "new-parent"));
await test("a recovered parent clear keeps the canonical source type distinct", (t) => checkRecoveredParent(t, null));

await test("undoing a recovered custom-property edit removes the stale change from submission", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("notes", "Text", { format: "Single line" })]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const persisted = {
    id: "saved",
    project_id: "first",
    type_id: "task",
    name: "Existing",
    description_html: "<p></p>",
    custom_properties: { notes: "old" },
  };
  const saved = [];
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        persistedData: persisted,
        data: { ...persisted, custom_properties: { notes: "new" } },
        onSubmit: async (value) => saved.push(value),
      })
    )
  );
  assert.equal(customControl(env.container, "notes").value, "new");
  await change(customControl(env.container, "notes"), "old");
  await submit(env.container);
  assert.equal(saved.length, 1);
  assert.deepEqual(saved[0].custom_properties, {});
});

await test("a failed former-type schema can be retried without losing edited values", async (t) => {
  const env = await setup(t);
  enableTypeFields((input) =>
    input.type_id === "bug" ? [] : [propertyDefinition("notes", "Text", { format: "Single line" })]
  );
  const schema = state.schema;
  let failSource = false;
  state.schema = async (input) => {
    if (failSource && input.type_id === "task") throw new Error("Source unavailable");
    return schema(input);
  };
  const saved = await editSavedNotes(env);
  failSource = true;
  await chooseWorkItemType(env.container, "bug");
  const retry = [...env.container.querySelectorAll("button")].find((button) => button.textContent === "Retry");
  assert.ok(retry, "source failure exposes retry even when destination has no fields");
  await submit(env.container);
  assert.equal(saved.length, 0);
  failSource = false;
  await act(async () => retry.click());
  await submit(env.container);
  assert.equal(saved.length, 1);
  assert.equal(saved[0].custom_properties.notes, "B");
});
