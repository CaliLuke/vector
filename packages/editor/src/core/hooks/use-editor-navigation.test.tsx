import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { act, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { expect, test } from "vitest";
import { useEditorNavigation } from "./use-editor-navigation";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

test("navigation keeps its extensions stable and directs shortcuts to the current editor", () => {
  const host = document.createElement("div");
  const root = createRoot(host);
  let current: ReturnType<typeof useEditorNavigation> | null = null;
  const onReady = (navigation: ReturnType<typeof useEditorNavigation>) => {
    current = navigation;
  };
  function Harness() {
    const navigation = useEditorNavigation();
    useEffect(() => onReady(navigation));
    return null;
  }
  const read = (): ReturnType<typeof useEditorNavigation> => {
    if (!current) throw new Error("Navigation did not mount");
    return current;
  };
  const editors: Editor[] = [];
  try {
    act(() => root.render(<Harness />));
    const navigation = read();
    const title = new Editor({
      extensions: [StarterKit.configure({ gapcursor: false }), navigation.titleNavigationExtension],
      content: "<p>Title</p>",
    });
    const first = new Editor({
      extensions: [StarterKit.configure({ gapcursor: false }), navigation.mainNavigationExtension],
      content: "<p>First</p>",
    });
    const second = new Editor({
      extensions: [StarterKit.configure({ gapcursor: false }), navigation.mainNavigationExtension],
      content: "<p>Second</p>",
    });
    editors.push(title, first, second);
    first.commands.setTextSelection(3);
    second.commands.setTextSelection(4);
    navigation.setMainEditor(first);
    expect(title.commands.keyboardShortcut("ArrowDown")).toBe(true);
    expect(first.state.selection.from).toBe(1);
    act(() => root.render(<Harness />));
    expect(read()).toBe(navigation);
    navigation.setMainEditor(second);
    expect(title.commands.keyboardShortcut("ArrowDown")).toBe(true);
    expect(second.state.selection.from).toBe(1);
    navigation.setTitleEditor(title);
    first.commands.setTextSelection(1);
    title.commands.setTextSelection(2);
    first.commands.keyboardShortcut("ArrowUp");
    expect(title.state.selection.from).toBe(6);
    second.commands.setTextSelection(1);
    title.commands.setTextSelection(2);
    second.commands.keyboardShortcut("ArrowLeft");
    expect(title.state.selection.from).toBe(6);
    first.commands.setTextSelection(3);
    title.commands.setTextSelection(2);
    first.commands.keyboardShortcut("ArrowUp");
    expect(title.state.selection.from).toBe(2);
    navigation.setTitleEditor(null);
    first.commands.setTextSelection(1);
    first.commands.keyboardShortcut("ArrowLeft");
    expect(title.state.selection.from).toBe(2);
    navigation.setMainEditor(null);
    second.commands.setTextSelection(4);
    title.commands.keyboardShortcut("ArrowDown");
    expect(second.state.selection.from).toBe(4);
  } finally {
    editors.forEach((editor) => editor.destroy());
    act(() => root.unmount());
  }
});
