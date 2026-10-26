import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { setup, state, issueProps } from "./helpers/issue-form-harness.mjs";
import { submitForm } from "./helpers/native-forms.mjs";
import { registerIssueRealEditor, setupRealEditor } from "./helpers/issue-real-editor.mjs";
registerIssueRealEditor();

await test("failed submission preserves custom rich text history and focuses the live editor", async (t) => {
  const env = await setupRealEditor(t, setup, state);
  state.types = { enabled: true, types: [{ id: "task", name: "Task", available: true, active: true }] };
  state.schema = async () => ({
    enabled: true,
    type_id: "task",
    eligible_type_ids: ["task"],
    selection_reason: "",
    revision: 1,
    computed: {},
    members: [],
    releases: [],
    properties: [
      {
        id: "notes",
        name: "Notes",
        title: "Notes",
        kind: "Rich text",
        context_id: "",
        configuration: { active: true, required: false, default_value: null },
      },
    ],
  });
  let rejectSave;
  const saving = new Promise((_, reject) => {
    rejectSave = reject;
  });
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        data: {
          name: "New",
          description_html: "<p></p>",
          project_id: "first",
          type_id: "task",
          custom_properties: { notes: "<p>Original</p>" },
        },
        onSubmit: () => saving,
      })
    )
  );
  const editor = [...state.actualEditors.entries()].find(([id]) => id.endsWith("-notes"))?.[1];
  assert.ok(editor, "custom field mounts the production editor hook");
  await act(async () => editor.commands.insertContentAt(1, "Edited "));
  assert.equal(editor.getText(), "Edited Original");
  await submitForm(env.container);
  assert.ok(env.container.querySelector("fieldset[inert]"), "saving locks the form");
  assert.equal(editor.isDestroyed, false, "submission must not recreate TipTap");
  await act(async () => rejectSave({ custom_properties: { notes: "Review these notes." } }));
  // TipTap focus schedules a browser animation frame.
  await act(async () => new Promise((resolve) => requestAnimationFrame(resolve)));
  assert.equal(editor.isDestroyed, false);
  assert.equal(document.activeElement, editor.view.dom);
  assert.equal(editor.view.dom.getAttribute("aria-invalid"), "true");
  await act(async () => editor.commands.undo());
  assert.equal(editor.getText(), "Original", "the edit remains undoable after the rejected save");
});
