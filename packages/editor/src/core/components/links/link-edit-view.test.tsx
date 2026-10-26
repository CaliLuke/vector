/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Editor } from "@tiptap/core";
import { StarterKit as TiptapStarterKit } from "@tiptap/starter-kit";
import { act, StrictMode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { CustomLinkExtension } from "@/extensions/custom-link";
import { LinkEditView } from "./link-edit-view";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

const host = document.createElement("div");
let root: Root;
let editor: Editor;

function makeEditor(href: string) {
  return new Editor({
    element: document.createElement("div"),
    extensions: [TiptapStarterKit.configure({ link: false }), CustomLinkExtension],
    content: {
      type: "doc",
      content: [
        {
          type: "paragraph",
          content: [
            {
              type: "text",
              text: "linked",
              marks: [
                { type: "bold" },
                {
                  type: "link",
                  attrs: {
                    href,
                    target: "_self",
                    rel: "nofollow",
                    class: "kept-class",
                    title: "kept title",
                  },
                },
              ],
            },
          ],
        },
      ],
    },
  });
}

function renderLink(closeLinkView = vi.fn(), url = "https://old.example") {
  const node = editor.state.doc.nodeAt(1);
  if (!node) throw new Error("Test link is missing");
  root.render(
    <LinkEditView
      viewProps={{
        editor,
        from: 1,
        to: 1 + node.nodeSize,
        url,
        text: node.text ?? "",
        closeLinkView,
      }}
    />
  );
  return closeLinkView;
}

function linkMark() {
  return editor.state.doc.nodeAt(1)?.marks.find((mark) => mark.type.name === "link");
}

function setInputValue(input: HTMLInputElement, value: string) {
  const descriptor = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value");
  descriptor?.set?.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
}

async function submit() {
  const button = host.querySelector<HTMLButtonElement>('button[type="submit"]');
  if (!button) throw new Error("Save link button is missing");
  await act(async () => {
    button.click();
  });
}

beforeEach(() => {
  document.body.append(host);
  root = createRoot(host);
});

afterEach(async () => {
  await act(async () => root.unmount());
  editor?.destroy();
  host.replaceChildren();
  host.remove();
});

it("keeps the draft and editor unchanged when URL validation fails", async () => {
  editor = makeEditor("https://old.example");
  const close = vi.fn();
  await act(async () => {
    renderLink(close);
  });
  const input = host.querySelector<HTMLInputElement>('input[name="url"]');
  if (!input) throw new Error("URL input is missing");

  await act(async () => setInputValue(input, "javascript:alert(1)"));
  await submit();

  expect(editor.getText()).toBe("linked");
  expect(linkMark()?.attrs.href).toBe("https://old.example");
  expect(host.textContent).toContain("Enter a valid URL");
  expect(close).not.toHaveBeenCalled();
  expect(input.value).toBe("javascript:alert(1)");
});

it("updates text and URL atomically while retaining link attributes and other marks", async () => {
  editor = makeEditor("https://old.example");
  const close = vi.fn();
  await act(async () => {
    renderLink(close);
  });
  const urlInput = host.querySelector<HTMLInputElement>('input[name="url"]');
  const textInput = host.querySelector<HTMLInputElement>('input[name="text"]');
  if (!urlInput || !textInput) throw new Error("Link edit inputs are missing");

  await act(async () => {
    setInputValue(urlInput, "new.example/path");
    setInputValue(textInput, "updated link");
  });
  await submit();

  const textNode = editor.state.doc.nodeAt(1);
  expect(textNode?.text).toBe("updated link");
  expect(textNode?.marks.map((mark) => mark.type.name).toSorted()).toEqual(["bold", "link"]);
  expect(textNode?.marks.find((mark) => mark.type.name === "link")?.attrs).toMatchObject({
    href: "https://new.example/path",
    target: "_self",
    rel: "nofollow",
    class: "kept-class",
    title: "kept title",
  });
  expect(editor.state.selection.from).toBe(1);
  expect(editor.state.selection.to).toBe(1 + "updated link".length);
  expect(close).toHaveBeenCalledOnce();
});

it("updates only the URL without moving the editor selection", async () => {
  editor = makeEditor("https://old.example");
  const close = vi.fn();
  await act(async () => {
    renderLink(close);
    editor.commands.setTextSelection(3);
  });
  const input = host.querySelector<HTMLInputElement>('input[name="url"]');
  if (!input) throw new Error("URL input is missing");
  await act(async () => setInputValue(input, "new.example"));

  await submit();

  expect(editor.getText()).toBe("linked");
  expect(linkMark()?.attrs.href).toBe("https://new.example");
  expect(editor.state.selection.from).toBe(3);
  expect(editor.state.selection.to).toBe(3);
  expect(close).toHaveBeenCalledOnce();
});

it("rejects a stale link range and keeps the user's draft", async () => {
  editor = makeEditor("https://old.example");
  const close = vi.fn();
  await act(async () => {
    renderLink(close);
  });
  const urlInput = host.querySelector<HTMLInputElement>('input[name="url"]');
  if (!urlInput) throw new Error("URL input is missing");
  await act(async () => setInputValue(urlInput, "https://new.example"));
  await act(async () => {
    editor.commands.setContent("<p>replacement content</p>");
  });

  await submit();

  expect(editor.getText()).toBe("replacement content");
  expect(host.textContent).toContain("This link changed. Close the editor and try again.");
  expect(urlInput.value).toBe("https://new.example");
  expect(close).not.toHaveBeenCalled();
});

it("does not remove an empty link during StrictMode effect replay", async () => {
  editor = makeEditor("");
  const node = editor.state.doc.nodeAt(1);
  if (!node) throw new Error("Test link is missing");
  const closeLinkView = vi.fn();
  await act(async () => {
    root.render(
      <StrictMode>
        <LinkEditView
          viewProps={{
            editor,
            from: 1,
            to: 1 + node.nodeSize,
            url: "",
            text: node.text ?? "",
            closeLinkView,
          }}
        />
      </StrictMode>
    );
  });

  expect(linkMark()?.attrs.href).toBe("");
  expect(closeLinkView).not.toHaveBeenCalled();
});
