import { Editor, Extension } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import type { SuggestionProps } from "@tiptap/suggestion";
import { afterEach, expect, test, vi } from "vitest";
import { emojiSuggestion } from "./suggestion";
import { createSuggestionKeyHandlers, getNextValidIndex } from "@/helpers/tippy";

const editors: Editor[] = [];
afterEach(() => {
  for (const editor of editors.splice(0)) editor.destroy();
});

function fixture() {
  const storage = {
    emojis: [{ name: "smile", emoji: "😀", shortcodes: ["smile"], tags: ["happy"] }],
    forceOpen: true,
  };
  const activeMenus = new Set(["emoji"]);
  const editor = new Editor({
    extensions: [
      StarterKit,
      Extension.create({ name: "emoji", addStorage: () => storage }),
      Extension.create({
        name: "menuCommands",
        addCommands: () => ({
          removeActiveDropbarExtension: (name) => () => {
            activeMenus.delete(name);
            return true;
          },
        }),
      }),
    ],
    content: "<p>Text</p>",
  });
  editors.push(editor);
  const props: SuggestionProps = {
    editor,
    range: { from: 1, to: 1 },
    query: "sm",
    text: ":sm",
    items: storage.emojis,
    command: () => {},
    decorationNode: null,
    clientRect: null,
    placement: "bottom-start",
    offset: { mainAxis: 4, crossAxis: 0 },
    flip: true,
    loading: false,
    floatingUi: { placement: "bottom-start", strategy: "fixed", middleware: [] },
    mount: (element) => {
      document.body.append(element);
      return () => element.remove();
    },
  };
  const renderer = emojiSuggestion.render?.();
  if (!renderer) throw new Error("Emoji suggestion renderer is missing");
  return { editor, props, activeMenus, renderer };
}

test("Escape closes a forced emoji menu using its active editor and tolerates a subsequent exit", () => {
  const { editor, props, activeMenus, renderer } = fixture();
  renderer.onStart?.(props);
  const event = new KeyboardEvent("keydown", { key: "Escape", cancelable: true });
  expect(renderer.onKeyDown?.({ view: editor.view, range: props.range, event })).toBe(true);
  expect(event.defaultPrevented).toBe(true);
  expect(editor.storage.emoji).toMatchObject({ forceOpen: false });
  expect(activeMenus.size).toBe(0);
  expect(() => renderer.onExit?.(props)).not.toThrow();
});

test("Escape before a suggestion starts and updates after it closes are harmless", () => {
  const { editor, props, renderer } = fixture();
  expect(
    renderer.onKeyDown?.({
      view: editor.view,
      range: props.range,
      event: new KeyboardEvent("keydown", { key: "Escape" }),
    })
  ).toBe(true);
  expect(() => renderer.onUpdate?.(props)).not.toThrow();
});

test("emoji search returns only matching existing entries and defaults omit unavailable names", async () => {
  const { editor } = fixture();
  expect(await emojiSuggestion.items?.({ editor, signal: new AbortController().signal, query: "" })).toEqual(
    editor.storage.emoji.emojis
  );
  expect(await emojiSuggestion.items?.({ editor, signal: new AbortController().signal, query: "HA" })).toEqual(
    editor.storage.emoji.emojis
  );
  expect(await emojiSuggestion.items?.({ editor, signal: new AbortController().signal, query: "missing" })).toEqual([]);
});

test("suggestion navigation delegates handled keys and removes the menu element on exit", () => {
  const { editor } = fixture();
  const element = document.createElement("div");
  document.body.append(element);
  const close = vi.fn();
  const handlers = createSuggestionKeyHandlers({
    getComponent: () => ({ element, ref: { onKeyDown: ({ event }) => event.key === "ArrowDown" } }),
    close,
  });
  const down = new KeyboardEvent("keydown", { key: "ArrowDown", cancelable: true });
  expect(handlers.onKeyDown({ event: down })).toBe(true);
  expect(down.defaultPrevented).toBe(true);
  expect(handlers.onKeyDown({ event: new KeyboardEvent("keydown", { key: "x" }) })).toBe(false);
  handlers.onExit({ editor });
  expect(element.isConnected).toBe(false);
  expect(close).toHaveBeenCalledWith(editor);
});

test.each([
  ["ArrowDown", 0, 0, { section: 0, item: 1 }],
  ["ArrowDown", 0, 1, { section: 2, item: 0 }],
  ["ArrowDown", 2, 0, { section: 0, item: 0 }],
  ["ArrowUp", 0, 0, { section: 2, item: 0 }],
  ["ArrowUp", 2, 0, { section: 0, item: 1 }],
  ["ArrowDown", 1, 0, { section: 0, item: 0 }],
  ["ArrowUp", 1, 0, { section: 2, item: 0 }],
] as const)("suggestion navigation %s from %s:%s skips empty sections and wraps", (key, section, item, expected) => {
  expect(
    getNextValidIndex({
      event: new KeyboardEvent("keydown", { key }),
      sections: [{ items: ["first", "second"] }, { items: [] }, { items: ["last"] }],
      selectedIndex: { section, item },
    })
  ).toEqual(expected);
});

test("suggestion navigation with no available items has no selectable index", () => {
  expect(
    getNextValidIndex({
      event: new KeyboardEvent("keydown", { key: "ArrowUp" }),
      sections: [{ items: [] }],
      selectedIndex: { section: 0, item: 0 },
    })
  ).toBeUndefined();
});
