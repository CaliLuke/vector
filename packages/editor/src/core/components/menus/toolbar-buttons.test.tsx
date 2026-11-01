import { act, StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { renderToString } from "react-dom/server";
import { expect, test } from "vitest";
import { TOOLBAR_ITEMS } from "@/constants/common";
import type { ToolbarMenuItem } from "@/constants/common";
import { useEditorToolbar } from "@/hooks/use-editor-toolbar";
import type { EditorRefApi } from "@/types";
import { EditorToolbarButtons } from "./toolbar-buttons";

class ToolbarEditor {
  listeners = new Set<() => void>();
  active = new Set<string>();
  ready = false;
  beforeSubscribe?: () => void;
  isMenuItemActive: EditorRefApi["isMenuItemActive"] = ({ itemKey }) => this.active.has(itemKey);
  isEditorReadyToDiscard = () => this.ready;
  onStateChange = (listener: () => void) => {
    this.beforeSubscribe?.();
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };
  notify() {
    for (const listener of this.listeners) listener();
  }
}

const noop = () => {};
function Toolbar({
  editor,
  executeCommand = noop,
}: {
  editor: ToolbarEditor | null;
  executeCommand?: (item: ToolbarMenuItem) => void;
}) {
  const { activeStates, isReadyToDiscard } = useEditorToolbar(editor, TOOLBAR_ITEMS.lite);
  return (
    <>
      <EditorToolbarButtons
        groups={TOOLBAR_ITEMS.lite}
        activeStates={activeStates}
        executeCommand={executeCommand}
        variant="comment"
      />
      <button type="submit" disabled={!isReadyToDiscard}>
        Submit
      </button>
    </>
  );
}

test("formatting and upload-only changes update controls; replacing the editor releases its subscription", () => {
  const host = document.createElement("div");
  const root = createRoot(host);
  const first = new ToolbarEditor();
  const second = new ToolbarEditor();
  let selected: ToolbarMenuItem | undefined;
  const executeCommand = (item: ToolbarMenuItem) => {
    selected = item;
  };
  const render = (editor: ToolbarEditor | null) =>
    act(() =>
      root.render(
        <StrictMode>
          <Toolbar editor={editor} executeCommand={executeCommand} />
        </StrictMode>
      )
    );
  try {
    render(first);
    expect(first.listeners.size).toBe(1);
    const bold = host.querySelector<HTMLButtonElement>('button[aria-label="Bold"]')!;
    const submit = host.querySelector<HTMLButtonElement>('button[type="submit"]')!;
    expect(bold.getAttribute("aria-pressed")).toBe("false");
    expect(submit.disabled).toBe(true);
    act(() => {
      first.active.add("bold");
      first.notify();
    });
    expect(bold.getAttribute("aria-pressed")).toBe("true");
    act(() => {
      first.ready = true;
      first.notify();
    });
    expect(submit.disabled).toBe(false);
    act(() => first.notify());
    expect(host.querySelector('button[aria-label="Bold"]')).toBe(bold);
    act(() => bold.click());
    expect(selected).toBe(TOOLBAR_ITEMS.lite.basic.find((item) => item.itemKey === "bold"));
    render(second);
    expect(first.listeners.size).toBe(0);
    expect(second.listeners.size).toBe(1);
    expect(bold.getAttribute("aria-pressed")).toBe("false");
    expect(submit.disabled).toBe(true);
    render(null);
    expect(second.listeners.size).toBe(0);
    expect(submit.disabled).toBe(true);
  } finally {
    act(() => root.unmount());
  }
});

test("changes between render and subscription are observed and unmount releases the active editor", () => {
  const host = document.createElement("div");
  const root = createRoot(host);
  const editor = new ToolbarEditor();
  editor.beforeSubscribe = () => {
    editor.ready = true;
  };
  try {
    act(() => root.render(<Toolbar editor={editor} />));
    expect(host.querySelector<HTMLButtonElement>('button[type="submit"]')!.disabled).toBe(false);
    expect(editor.listeners.size).toBe(1);
  } finally {
    act(() => root.unmount());
  }
  expect(editor.listeners.size).toBe(0);
});

test.each(["publicComment", "sticky"] as const)(
  "%s toolbar keeps command order and selected button semantics",
  (variant) => {
    const host = document.createElement("div");
    const root = createRoot(host);
    try {
      act(() =>
        root.render(
          <EditorToolbarButtons
            groups={TOOLBAR_ITEMS.sticky}
            activeStates={{ bold: true }}
            executeCommand={noop}
            variant={variant}
          />
        )
      );
      expect(Array.from(host.querySelectorAll("button"), (button) => button.getAttribute("aria-label"))).toEqual(
        Object.values(TOOLBAR_ITEMS.sticky)
          .flat()
          .map((item) => item.name)
      );
      expect(host.querySelector('button[aria-label="Bold"]')!.getAttribute("aria-pressed")).toBe("true");
      expect(host.querySelector('button[aria-label="Italic"]')!.getAttribute("aria-pressed")).toBe("false");
    } finally {
      act(() => root.unmount());
    }
  }
);

test("server rendering uses a stable disabled submission snapshot without subscribing", () => {
  const editor = new ToolbarEditor();
  editor.ready = true;
  expect(renderToString(<Toolbar editor={editor} />)).toContain('type="submit" disabled=""');
  expect(editor.listeners.size).toBe(0);
});
