/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { act } from "react";
import { createRoot } from "react-dom/client";
import { Editor } from "@tiptap/core";
import StarterKitExtension from "@tiptap/starter-kit";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { CustomLinkExtension } from "@/extensions/custom-link/extension";
import * as editorCommands from "@/helpers/editor-commands";
import { BubbleMenuLinkSelector } from "./link-selector";

let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let editor: Editor;
const rangeRectDescriptors = {
  getClientRects: Object.getOwnPropertyDescriptor(Range.prototype, "getClientRects"),
  getBoundingClientRect: Object.getOwnPropertyDescriptor(Range.prototype, "getBoundingClientRect"),
};

beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const emptyRect = { x: 0, y: 0, top: 0, right: 0, bottom: 0, left: 0, width: 0, height: 0, toJSON: () => ({}) };
  Object.defineProperty(Range.prototype, "getClientRects", { configurable: true, value: () => [] });
  Object.defineProperty(Range.prototype, "getBoundingClientRect", { configurable: true, value: () => emptyRect });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  editor = new Editor({
    element: document.createElement("div"),
    extensions: [StarterKitExtension.configure({ link: false }), CustomLinkExtension],
    content: "<p>first second</p>",
  });
  document.body.append(editor.view.dom);
});

afterEach(async () => {
  await act(async () => root.unmount());
  editor.destroy();
  host.remove();
  vi.restoreAllMocks();
  for (const [name, descriptor] of Object.entries(rangeRectDescriptors)) {
    if (descriptor) Object.defineProperty(Range.prototype, name, descriptor);
    else Reflect.deleteProperty(Range.prototype, name);
  }
});

async function openMenu() {
  const trigger = host.querySelector("button");
  if (!trigger) throw new Error("Link menu trigger is missing");
  await act(async () => trigger.dispatchEvent(new MouseEvent("click", { bubbles: true })));
  const input = document.body.querySelector<HTMLInputElement>('input[placeholder="Enter or paste a link"]');
  if (!input) throw new Error("Link URL input is missing");
  return input;
}

async function setInputValue(input: HTMLInputElement, value: string) {
  await act(async () => {
    const descriptor = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value");
    descriptor?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

it("validates a URL through the native form and applies it to the current editor selection", async () => {
  editor.commands.setTextSelection({ from: 1, to: 6 });
  await act(async () => root.render(<BubbleMenuLinkSelector editor={editor} />));
  const input = await openMenu();
  await setInputValue(input, "example.com");
  const enter = new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true });
  await act(async () => input.dispatchEvent(enter));

  expect(enter.defaultPrevented).toBe(true);
  expect(editor.getAttributes("link").href).toBe("https://example.com");
  expect(editor.state.selection.from).toBe(1);
  expect(editor.state.selection.to).toBe(6);
  expect(document.body.querySelector('input[placeholder="Enter or paste a link"]')).toBeNull();
});

it("shows native validation feedback and retains the rejected URL draft", async () => {
  editor.commands.setTextSelection({ from: 1, to: 6 });
  await act(async () => root.render(<BubbleMenuLinkSelector editor={editor} />));
  const input = await openMenu();
  expect(document.activeElement).toBe(input);
  await act(async () => input.form?.requestSubmit());
  expect(document.body.querySelector('[role="alert"]')?.textContent).toBe("Please enter a valid URL");
  await setInputValue(input, "javascript:alert(1)");
  expect(document.body.querySelector('[role="alert"]')).toBeNull();
  await act(async () => input.form?.requestSubmit());

  expect(document.body.querySelector('[role="alert"]')?.textContent).toBe("Please enter a valid URL");
  expect(input.value).toBe("javascript:alert(1)");
  expect(editor.getAttributes("link").href).toBeUndefined();
});

it("keeps a failed editor command unsuccessful and lets the user retry the retained draft", async () => {
  editor.commands.setTextSelection({ from: 1, to: 6 });
  await act(async () => root.render(<BubbleMenuLinkSelector editor={editor} />));
  const input = await openMenu();
  await setInputValue(input, "https://retry.example");
  const setLink = vi.spyOn(editorCommands, "setLinkEditor").mockImplementation(() => {
    throw new Error("Editor is temporarily unavailable");
  });

  await act(async () => input.form?.requestSubmit());
  expect(document.body.querySelector('[role="alert"]')?.textContent).toBe(
    "Could not apply the link. Please try again."
  );
  expect(input.value).toBe("https://retry.example");
  expect(editor.getAttributes("link").href).toBeUndefined();

  setLink.mockRestore();
  await act(async () => input.form?.requestSubmit());
  expect(editor.getAttributes("link").href).toBe("https://retry.example");
  expect(document.body.querySelector('[role="alert"]')).toBeNull();
});

it("loads the URL for the newly selected link each time the menu opens", async () => {
  editor.commands.setTextSelection({ from: 1, to: 6 });
  editor.chain().setLink({ href: "https://first.example" }).run();
  editor.commands.setTextSelection({ from: 7, to: 13 });
  await act(async () => root.render(<BubbleMenuLinkSelector editor={editor} />));

  const input = await openMenu();
  expect(input.value).toBe("");
  const trigger = host.querySelector("button");
  await act(async () => trigger?.dispatchEvent(new MouseEvent("click", { bubbles: true })));
  expect(document.body.querySelector('input[placeholder="Enter or paste a link"]')).toBeNull();

  editor.commands.setTextSelection({ from: 1, to: 6 });
  await act(async () => root.render(<BubbleMenuLinkSelector editor={editor} />));
  const reopenedInput = await openMenu();
  expect(reopenedInput.value).toBe("https://first.example");
});

it("removes an active link while preserving the editor selection", async () => {
  editor.commands.setTextSelection({ from: 1, to: 6 });
  editor.chain().setLink({ href: "https://first.example" }).run();
  editor.commands.setTextSelection({ from: 1, to: 6 });
  await act(async () => root.render(<BubbleMenuLinkSelector editor={editor} />));
  await openMenu();
  await act(async () => document.body.querySelector<HTMLButtonElement>('button[aria-label="Remove link"]')?.click());

  expect(editor.getAttributes("link").href).toBeUndefined();
  expect(editor.state.selection.from).toBe(1);
});

it("resets the open draft when its editor instance changes", async () => {
  editor.commands.setTextSelection({ from: 1, to: 6 });
  await act(async () => root.render(<BubbleMenuLinkSelector editor={editor} />));
  const input = await openMenu();
  await setInputValue(input, "https://unsaved.example");
  const previousEditor = editor;
  editor = new Editor({
    element: document.createElement("div"),
    extensions: [StarterKitExtension.configure({ link: false }), CustomLinkExtension],
    content: '<p><a href="https://replacement.example">replacement</a></p>',
  });
  editor.commands.setTextSelection({ from: 1, to: 12 });
  try {
    await act(async () => root.render(<BubbleMenuLinkSelector editor={editor} />));
    const nextInput = document.body.querySelector<HTMLInputElement>('input[aria-label="Link URL"]');
    expect(nextInput?.value).toBe("https://replacement.example");
    expect(document.activeElement).toBe(nextInput);
    expect(previousEditor.getAttributes("link").href).toBeUndefined();
  } finally {
    previousEditor.destroy();
  }
});
