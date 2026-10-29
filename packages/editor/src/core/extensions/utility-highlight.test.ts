import { Editor } from "@tiptap/core";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createEditorHookProps } from "../../../tests/helpers/editor-hook-props";
import { nodeHighlightPluginKey } from "../plugins/highlight";
import { CustomStarterKitExtension } from "./starter-kit";
import { CustomCodeInlineExtension } from "./code-inline";
import { UniqueID } from "./unique-id/extension";
import { UtilityExtension } from "./utility";

let editor: Editor | undefined;

afterEach(() => {
  editor?.destroy();
  editor = undefined;
});

function createEditor(editable = true) {
  const props = createEditorHookProps({ editable });
  editor = new Editor({
    editable,
    extensions: [
      CustomStarterKitExtension({ enableHistory: true }),
      CustomCodeInlineExtension,
      UniqueID.configure({ updateDocument: false }),
      UtilityExtension({ ...props, isEditable: editable, isTouchDevice: false }),
    ],
    content: {
      type: "doc",
      content: [
        { type: "paragraph", attrs: { id: "first" }, content: [{ type: "text", text: "First block" }] },
        { type: "paragraph", attrs: { id: "target" }, content: [{ type: "text", text: "Linked block" }] },
      ],
    },
  });
  return editor;
}

function highlight(current: Editor, nodeId: string | null) {
  current.view.dispatch(current.state.tr.setMeta(nodeHighlightPluginKey, { nodeId }));
}

function highlightedBlockIds(current: Editor) {
  return Array.from(current.view.dom.querySelectorAll("[data-node-highlighted]"), (node) =>
    node.getAttribute("data-id")
  );
}

describe("block-link highlights in the editor utility extension", () => {
  it.each([true, false])("highlights and clears a linked block without editing content (editable=%s)", (editable) => {
    const current = createEditor(editable);
    const before = current.getJSON();
    const update = vi.fn();
    current.on("update", update);

    highlight(current, "target");
    expect(highlightedBlockIds(current)).toEqual(["target"]);
    current.commands.setTextSelection(2);
    expect(highlightedBlockIds(current)).toEqual(["target"]);

    highlight(current, null);
    expect(highlightedBlockIds(current)).toEqual([]);
    expect(current.getJSON()).toEqual(before);
    expect(update).not.toHaveBeenCalled();
  });

  it("keeps the highlight on the linked block when edits move it and removes it when the block is deleted", () => {
    const current = createEditor();
    highlight(current, "target");
    expect(highlightedBlockIds(current)).toEqual(["target"]);
    current.commands.insertContentAt(0, {
      type: "paragraph",
      attrs: { id: "inserted" },
      content: [{ type: "text", text: "Inserted before the linked block" }],
    });
    expect(highlightedBlockIds(current)).toEqual(["target"]);

    let target: { from: number; to: number } | undefined;
    current.state.doc.descendants((node, pos) => {
      if (node.attrs.id === "target") target = { from: pos, to: pos + node.nodeSize };
    });
    if (!target) throw new Error("Linked block is missing from the test document");
    current.commands.deleteRange(target);
    expect(highlightedBlockIds(current)).toEqual([]);
    expect(current.getText()).not.toContain("Linked block");
  });

  it("does not decorate another block when the requested block is absent", () => {
    const current = createEditor();
    highlight(current, "target");
    expect(highlightedBlockIds(current)).toEqual(["target"]);
    highlight(current, "missing");
    expect(highlightedBlockIds(current)).toEqual([]);
    highlight(current, "first");
    expect(highlightedBlockIds(current)).toEqual(["first"]);
  });
});
