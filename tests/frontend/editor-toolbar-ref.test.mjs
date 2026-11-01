import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebTestModules, reactModuleUrl } from "./helpers/web-test-modules.mjs";

const editorUrl = new URL("../../packages/editor/dist/index.js", import.meta.url).href;
registerWebTestModules({
  "@plane/editor": `
    export * from ${JSON.stringify(editorUrl)};
    import React from ${JSON.stringify(reactModuleUrl)};
    export const LiteTextEditorWithRef = React.forwardRef(function EditorFixture(props, ref) {
      React.useImperativeHandle(ref, () => globalThis.toolbarEditorApi, []);
      return React.createElement('div', null, 'Editor');
    });`,
  "@/hooks/use-parse-editor-content":
    "const getEditorMetaData=()=>({file_assets:[],user_mentions:[]}); export const useParseEditorContent=()=>({getEditorMetaData});",
  "@/hooks/editor": "export const useEditorConfig=()=>({});",
  "@/components/editor/extensions": "export const DISABLED_EDITOR_EXTENSIONS=[];",
  "./embeds/mentions": "export const EditorMentionsRoot=()=>null;",
  "@/helpers/editor.helper": "export const getEditorFileHandlers=()=>({});",
});

for (const kind of ["sticky", "comment"]) {
  void test(`${kind} toolbar receives its editor at mount with a callback ref`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const commands = [];
    const received = [];
    globalThis.toolbarEditorApi = {
      executeMenuItemCommand: (command) => commands.push(command),
      isMenuItemActive: ({ itemKey }) => itemKey === "bold",
      isEditorReadyToDiscard: () => false,
      onStateChange: () => () => {},
    };
    t.after(() => {
      delete globalThis.toolbarEditorApi;
    });
    const Component =
      kind === "sticky"
        ? (await import("../../apps/web/core/components/editor/sticky-editor/editor.tsx")).StickyEditor
        : (await import("../../apps/space/components/editor/lite-text-editor.tsx")).LiteTextEditor;
    await env.render(
      React.createElement(Component, {
        ref: (value) => {
          received.push(value);
        },
        workspaceSlug: "acme",
        workspaceId: "workspace",
        anchor: "published",
        initialValue: "<p>Text</p>",
        editable: true,
        uploadFile: async () => "asset",
        handleColorChange: async () => {},
        handleDelete: () => {},
      })
    );
    assert.equal(received[0], globalThis.toolbarEditorApi);
    const bold = env.container.querySelector('button[aria-label="Bold"]');
    assert.ok(bold, "the real toolbar renders its bold control");
    assert.equal(bold.getAttribute("aria-pressed"), "true");
    await act(async () => bold.click());
    assert.deepEqual(commands, [{ itemKey: "bold" }]);
    await env.render(null);
    assert.equal(received.at(-1), null);
  });
}
