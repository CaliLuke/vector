import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { Bold } from "lucide-react";
import { expect, test } from "vitest";
import { TEXT_ALIGNMENT_ITEMS, TOOLBAR_ITEMS } from "@/constants/common";
import type { ToolbarMenuItem } from "@/constants/common";
import { getEditorMenuItem } from "@/components/menus/menu-items";
import type { EditorRefApi } from "@/types";
import { getEditorRefHelpers } from "./editor-ref";
import { executeToolbarCommand } from "./toolbar-command";

test("toolbar commands preserve their required selection, alignment, color and link arguments", () => {
  const received: Parameters<EditorRefApi["executeMenuItemCommand"]>[0][] = [];
  const execute: EditorRefApi["executeMenuItemCommand"] = (command) => {
    received.push(command);
  };
  const presentation = { renderKey: "test", name: "Test", icon: Bold, editors: [] };
  const items: ToolbarMenuItem[] = [
    ...Object.values(TOOLBAR_ITEMS.lite).flat(),
    ...TEXT_ALIGNMENT_ITEMS,
    { ...presentation, itemKey: "attachment", extraProps: { savedSelection: null } },
    { ...presentation, itemKey: "text-color", extraProps: { color: "green" } },
    { ...presentation, itemKey: "background-color", extraProps: { color: undefined } },
    { ...presentation, itemKey: "link", extraProps: { url: "https://example.com", text: "Example" } },
  ];
  for (const item of items) executeToolbarCommand(execute, item);
  for (const [index, item] of items.entries())
    expect(received[index]).toEqual({ itemKey: item.itemKey, ...item.extraProps });
});

test("editor command lookup retains command-specific state and mutation behavior", () => {
  const editor = new Editor({ extensions: [StarterKit], content: "<p>Draft</p>" });
  const api = getEditorRefHelpers({
    editor,
    provider: undefined,
    getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
  });
  try {
    editor.commands.setTextSelection({ from: 1, to: 6 });
    expect(api.isMenuItemActive({ itemKey: "bold" })).toBe(false);
    api.executeMenuItemCommand({ itemKey: "bold" });
    expect(api.isMenuItemActive({ itemKey: "bold" })).toBe(true);
    expect(editor.getHTML()).toBe("<p><strong>Draft</strong></p>");
    api.executeMenuItemCommand({ itemKey: "h2" });
    expect(api.isMenuItemActive({ itemKey: "h2" })).toBe(true);
    expect(api.isMenuItemActive({ itemKey: "h1" })).toBe(false);
    expect(api.isMenuItemActive({ itemKey: "external-embed" })).toBe(false);
    expect(getEditorMenuItem(null, "bold")).toBeUndefined();
  } finally {
    editor.destroy();
  }
});
