import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { EditorContent } from "@tiptap/react";
import { act, StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { CustomLinkExtension } from "@/extensions/custom-link";
import { TitleExtensions } from "@/extensions/title-extension";
import { EditorContainer } from "./editor-container";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

test("desktop title editors mount and unmount without link-preview commands", async () => {
  const editor = new Editor({ extensions: TitleExtensions, content: "<h1>Review notes</h1>" });
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  try {
    await act(async () =>
      root.render(
        <StrictMode>
          <EditorContainer
            editor={editor}
            displayConfig={{}}
            editorContainerClassName=""
            id="title"
            isTouchDevice={false}
          >
            <EditorContent editor={editor} />
          </EditorContainer>
        </StrictMode>
      )
    );
    expect(host.textContent).toContain("Review notes");
    await act(async () => editor.commands.insertContentAt(1, "Saved "));
    expect(editor.getText()).toBe("Saved Review notes");
    await act(async () => root.unmount());
    expect(host.textContent).toBe("");
  } finally {
    await act(async () => root.unmount());
    editor.destroy();
    host.remove();
  }
});

async function fixture(content: string, editable = true) {
  const editor = new Editor({
    extensions: [StarterKit.configure({ trailingNode: false, link: false }), CustomLinkExtension],
    content,
    editable,
    editorProps: { handleScrollToSelection: () => true },
  });
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  const childClick = vi.fn();
  await act(async () =>
    root.render(
      <EditorContainer
        editor={editor}
        displayConfig={{}}
        editorContainerClassName="p-4"
        id="test"
        isTouchDevice={false}
      >
        <EditorContent editor={editor} />
        <button type="button" onClick={childClick}>
          Child action
        </button>
      </EditorContainer>
    )
  );
  return {
    editor,
    host,
    childClick,
    close: async () => {
      await act(async () => root.unmount());
      editor.destroy();
      host.remove();
    },
  };
}

test("desktop body editors retain link previews when their schema supports them", async () => {
  const f = await fixture('<p><a href="https://example.com/notes">Notes</a></p>');
  try {
    const anchor = f.host.querySelector("a");
    if (!anchor) throw new Error("Missing editor link");
    await act(async () => anchor.focus());
    expect(f.host.textContent).toContain("https://example.com/notes");
    expect(f.editor.storage.link.isPreviewOpen).toBe(true);
    await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
    expect(f.editor.storage.link.isPreviewOpen).toBe(false);
    expect(f.editor.getHTML()).toContain('href="https://example.com/notes"');
  } finally {
    await f.close();
  }
});

test("the continue-writing button inserts a paragraph after a terminal heading and focuses it", async () => {
  const f = await fixture("<h2>Title</h2>");
  try {
    const trigger = f.host.querySelector<HTMLButtonElement>(
      'button[aria-label="Continue writing at end of document"]'
    )!;
    expect(trigger.type).toBe("button");
    await act(async () => {
      trigger.focus();
      trigger.click();
    });
    expect(f.editor.getHTML()).toBe("<h2>Title</h2><p></p>");
    await vi.waitFor(() => expect(f.editor.state.selection.$from.parent.type.name).toBe("paragraph"));
    await vi.waitFor(() => expect(f.editor.isFocused).toBe(true));
    await act(async () => {
      trigger.focus();
      trigger.click();
    });
    expect(f.editor.getHTML()).toBe("<h2>Title</h2><p></p>");
  } finally {
    await f.close();
  }
});

test("read-only documents have no continue-writing control and child actions retain their own behavior", async () => {
  const f = await fixture("<p>Draft</p>", false);
  try {
    expect(f.host.querySelector('[aria-label="Continue writing at end of document"]')).toBeNull();
    await act(async () => f.host.querySelector<HTMLButtonElement>("button")!.click());
    expect(f.childClick).toHaveBeenCalledOnce();
    expect(f.editor.getHTML()).toBe("<p>Draft</p>");
  } finally {
    await f.close();
  }
});

test("continue writing after an atomic block creates a text position without relying on a trailing-node plugin", async () => {
  const f = await fixture("<hr>");
  try {
    const trigger = f.host.querySelector<HTMLButtonElement>(
      'button[aria-label="Continue writing at end of document"]'
    )!;
    await act(async () => trigger.click());
    expect(f.editor.getHTML()).toBe("<hr><p></p>");
    expect(f.editor.state.selection.$from.parent.type.name).toBe("paragraph");
  } finally {
    await f.close();
  }
});
