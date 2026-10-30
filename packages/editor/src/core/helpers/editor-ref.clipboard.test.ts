/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { getEditorRefHelpers } from "./editor-ref";

let editor: Editor;
const originalCommand = Object.getOwnPropertyDescriptor(document, "execCommand");
let cleanupListeners: () => void;

beforeEach(() => {
  editor = new Editor({ extensions: [StarterKit], content: "<p>Draft</p>" });
  const copyListeners = vi.spyOn(document, "addEventListener");
  cleanupListeners = () => {
    // Also isolate failing regression runs: old source leaks this listener.
    for (const [name, listener] of copyListeners.mock.calls) {
      if (name === "copy") document.removeEventListener("copy", listener);
    }
  };
});

afterEach(() => {
  cleanupListeners();
  vi.restoreAllMocks();
  if (originalCommand) Object.defineProperty(document, "execCommand", originalCommand);
  else Reflect.deleteProperty(document, "execCommand");
  editor.destroy();
});

function copy(editorToCopy: Editor | null = editor) {
  getEditorRefHelpers({
    editor: editorToCopy,
    provider: undefined,
    getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
  }).copyMarkdownToClipboard();
}

function setCommand(command: () => boolean) {
  Object.defineProperty(document, "execCommand", { configurable: true, value: command });
}

function dispatchCopy() {
  const data = new Map<string, string>();
  const event = new Event("copy", { cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: { setData: (type: string, value: string) => data.set(type, value) },
  });
  document.dispatchEvent(event);
  return { data, event };
}

it("copies markdown and both rich-text formats only for the requested copy", () => {
  let copied: ReturnType<typeof dispatchCopy> | undefined;
  setCommand(() => {
    copied = dispatchCopy();
    return true;
  });
  copy();
  expect(copied?.data.get("text/plain")).toBe("Draft\n");
  expect(copied?.data.get("text/html")).toBe("<p>Draft</p>");
  expect(copied?.data.get("text/plane-editor-html")).toBe("<p>Draft</p>");
  expect(copied?.event.defaultPrevented).toBe(true);
  expect(dispatchCopy().data.size).toBe(0);
});

it("rejects a denied copy and leaves later copy events untouched", () => {
  setCommand(() => false);
  expect(copy).toThrow();
  expect(dispatchCopy().data.size).toBe(0);
});

it("does not report success when the editor is unavailable", () => {
  expect(() => copy(null)).toThrow();
  expect(dispatchCopy().data.size).toBe(0);
});

it("removes its listener when the browser command throws", () => {
  setCommand(() => {
    throw new Error("Browser denied clipboard access");
  });
  expect(copy).toThrow();
  const later = dispatchCopy();
  expect(later.data.size).toBe(0);
  expect(later.event.defaultPrevented).toBe(false);
});

it("does not report success when the command dispatches no copy event", () => {
  setCommand(() => true);
  expect(copy).toThrow();
  expect(dispatchCopy().data.size).toBe(0);
});

it("does not report success when the copy event has no writable clipboard", () => {
  setCommand(() => {
    document.dispatchEvent(new Event("copy", { cancelable: true }));
    return true;
  });
  expect(copy).toThrow();
  expect(dispatchCopy().data.size).toBe(0);
});
