import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act } from "./helpers/react.mjs";
import { setup, state } from "./helpers/issue-form-harness.mjs";
import { propertyDefinition } from "./helpers/issue-custom-fields.mjs";
import { submitForm } from "./helpers/native-forms.mjs";
import { registerIssueRealEditor, setupRealEditor } from "./helpers/issue-real-editor.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerIssueRealEditor();
registerHooks({
  resolve(specifier, context, next) {
    return resolvePackageStub(
      specifier,
      {
        "./use-inline-property-preview":
          "export const useInlinePropertyPreview=()=>({computed:{},error:false,changed:false});",
      },
      context,
      next
    );
  },
});

await test("inline rich text settles after transactions and responds to upload readiness", async (t) => {
  const env = await setupRealEditor(t, setup, state);
  state.actualEditorRenders = 0;
  state.actualEditorRenderLimit = 40;
  const { InlinePropertyForm } =
    await import("../../apps/web/core/components/work-item-types/inline-property-form.tsx");
  const commands = [];
  const snapshot = {
    source: {
      source_version: "source-1",
      item: { id: "item", project_id: "first", custom_properties: { notes: "<p>Original</p>" } },
    },
    schema: {
      enabled: true,
      revision: 4,
      properties: [propertyDefinition("notes", "Rich text")],
      computed: {},
      members: [],
      releases: [],
    },
  };
  await env.render(
    React.createElement(InlinePropertyForm, {
      scope: { workspaceSlug: "workspace", projectId: "first", issueId: "item" },
      snapshot,
      edit: {
        source: { isFetching: false },
        busy: false,
        conflict: false,
        save: async (command) => commands.push(command),
      },
      onClose: () => {},
      onAccessLost: () => {},
    })
  );
  const editor = [...state.actualEditors.values()][0];
  assert.ok(editor, "mounts the production editor hook");
  const save = () =>
    [...env.container.querySelectorAll("button")].find((button) => button.textContent === "Save properties");
  assert.equal(save().disabled, false);
  await act(async () => editor.commands.insertContentAt(1, "Edited "));
  assert.equal(editor.getText(), "Edited Original");
  await act(async () => {
    editor.storage.utility.uploadInProgress = true;
    editor.view.dispatch(editor.state.tr);
  });
  assert.equal(save().disabled, true, "uploads prevent premature save");
  await act(async () => {
    editor.storage.utility.uploadInProgress = false;
    editor.view.dispatch(editor.state.tr);
  });
  assert.equal(save().disabled, false, "completion enables save without remounting");
  await submitForm(env.container);
  assert.equal(commands.length, 1);
  assert.match(commands[0].changes.custom_properties.notes, /Edited Original/);
  assert.equal(editor.isDestroyed, false);
  assert.ok(state.actualEditorRenders < 40, "transaction notifications must settle");
});
