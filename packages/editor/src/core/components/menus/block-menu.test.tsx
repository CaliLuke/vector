import { UtilityExtension } from "@/extensions/utility";
import { createEditorHookProps } from "../../../../tests/helpers/editor-hook-props";
import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { BlockMenu } from "./block-menu";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
const menu = () =>
  [...document.querySelectorAll("button")].find((button) => button.textContent === "Delete")?.parentElement;

test("closing a block menu cancels its animation and outside dismissal releases the active menu", async () => {
  vi.useFakeTimers();
  const menuCommands = UtilityExtension({ ...createEditorHookProps(), isEditable: true, isTouchDevice: false });
  const editor = new Editor({ extensions: [StarterKit, menuCommands], content: "<p>Text</p>" });
  const activeMenus = editor.storage.utility.activeDropbarExtensions;
  const host = document.createElement("div");
  const trigger = document.createElement("button");
  trigger.id = "drag-handle";
  document.body.append(host, trigger);
  const root = createRoot(host);
  try {
    await act(async () => root.render(<BlockMenu editor={editor} />));
    await act(async () => trigger.click());
    expect(activeMenus.length).toBe(1);
    expect(menu()).toBeTruthy();
    await act(() => vi.advanceTimersByTimeAsync(50));
    await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
    expect(menu()).toBeUndefined();
    expect(activeMenus.length).toBe(0);
    await act(async () => trigger.click());
    await act(() => vi.advanceTimersByTimeAsync(20));
    expect(menu()?.classList.contains("opacity-0")).toBe(true);
    await act(() => vi.advanceTimersByTimeAsync(80));
    expect(menu()?.classList.contains("opacity-100")).toBe(true);
    await act(async () => document.body.dispatchEvent(new MouseEvent("pointerdown", { bubbles: true })));
    expect(menu()).toBeUndefined();
    expect(activeMenus.length).toBe(0);
    await act(async () => trigger.click());
    expect(activeMenus.length).toBe(1);
    await act(async () => root.render(null));
    expect(menu()).toBeUndefined();
    expect(activeMenus.length).toBe(0);
  } finally {
    act(() => root.unmount());
    editor.destroy();
    host.remove();
    trigger.remove();
    vi.useRealTimers();
  }
});
