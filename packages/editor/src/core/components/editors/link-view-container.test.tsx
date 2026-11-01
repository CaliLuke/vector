import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { CustomLinkExtension } from "@/extensions/custom-link";
import { LinkViewContainer } from "./link-view-container";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

function createLinkPreviewFixture() {
  const container = document.createElement("div");
  const mount = document.createElement("div");
  document.body.append(container, mount);
  const editor = new Editor({
    element: container,
    editorProps: { handleScrollToSelection: () => true },
    extensions: [StarterKit.configure({ link: false }), CustomLinkExtension],
    content: '<p><a href="https://example.com/page">Example</a></p>',
  });
  const root = createRoot(mount);
  return {
    container,
    mount,
    editor,
    root,
    async close() {
      await act(async () => root.unmount());
      editor.destroy();
      container.remove();
      mount.remove();
    },
  };
}

test("link preview teardown is safe after its owning editor is destroyed", async () => {
  const fixture = createLinkPreviewFixture();
  const { container, mount, editor, root } = fixture;
  try {
    await act(async () => root.render(<LinkViewContainer editor={editor} containerRef={{ current: container }} />));
    const anchor = container.querySelector("a");
    if (!anchor) throw new Error("Missing editor link");
    await act(async () => anchor.focus());
    expect(mount.textContent).toContain("https://example.com/page");
    vi.useFakeTimers();
    await act(async () => container.dispatchEvent(new MouseEvent("mouseleave", { relatedTarget: null })));
    await act(async () => editor.destroy());
    expect(editor.isDestroyed).toBe(true);
    await act(async () => vi.advanceTimersByTime(400));
    expect(mount.textContent).toBe("");
    await act(async () => root.unmount());
    expect(mount.textContent).toBe("");
  } finally {
    vi.useRealTimers();
    await fixture.close();
  }
});

test("link previews open from focus and hover, dismiss without document edits, and clean up timers", async () => {
  const fixture = createLinkPreviewFixture();
  const { container, mount, editor, root } = fixture;
  const transaction = vi.fn();
  editor.on("transaction", transaction);
  try {
    await act(async () => root.render(<LinkViewContainer editor={editor} containerRef={{ current: container }} />));
    const anchor = container.querySelector("a");
    if (!anchor) throw new Error("Missing editor link");
    await act(async () => anchor.focus());
    expect(mount.textContent).toContain("https://example.com/page");
    expect(editor.storage.link.isPreviewOpen).toBe(true);
    expect(anchor.hasAttribute("onpointerdown")).toBe(false);
    await act(async () => editor.commands.setLinkBubbleMenuOpen(true));
    expect(mount.textContent).toBe("");
    expect(editor.storage.link.isPreviewOpen).toBe(false);
    await act(async () => anchor.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
    expect(mount.textContent).toBe("");
    await act(async () => editor.commands.setLinkBubbleMenuOpen(false));
    await act(async () => anchor.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
    expect(mount.textContent).toContain("https://example.com/page");
    await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
    expect(mount.textContent).toBe("");
    expect(editor.storage.link.isPreviewOpen).toBe(false);
    expect(transaction).not.toHaveBeenCalled();
    await act(async () => anchor.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
    expect(document.activeElement).toBe(anchor);
    const edit = mount.querySelector<HTMLButtonElement>('button[aria-label="Edit link"]');
    if (!edit) throw new Error("Missing edit link control");
    await act(async () => edit.click());
    await vi.waitFor(() => expect(document.activeElement?.tagName).toBe("INPUT"));
    await act(async () =>
      document.activeElement?.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
    );
    await vi.waitFor(() => expect(document.activeElement).toBe(editor.view.dom));
    expect(mount.textContent).toBe("");
    await act(async () => anchor.focus());

    expect(transaction.mock.calls.every(([event]) => !event.transaction.docChanged)).toBe(true);
    vi.useFakeTimers();
    await act(async () => anchor.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
    await act(async () => container.dispatchEvent(new MouseEvent("mouseleave", { relatedTarget: null })));
    await act(async () => vi.advanceTimersByTime(400));
    expect(mount.textContent).toBe("");
    await act(async () => anchor.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
    await act(async () => container.dispatchEvent(new MouseEvent("mouseleave", { relatedTarget: null })));
    await act(async () => root.unmount());
    expect(editor.storage.link.isPreviewOpen).toBe(false);
    await act(async () => vi.advanceTimersByTime(400));
  } finally {
    vi.useRealTimers();
    await fixture.close();
  }
});
