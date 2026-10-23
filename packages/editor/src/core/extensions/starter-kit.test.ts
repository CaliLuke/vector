/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Editor } from "@tiptap/core";
import { afterEach, describe, expect, it } from "vitest";
import { CustomStarterKitExtension } from "./starter-kit";

const extensions = [CustomStarterKitExtension({ enableHistory: true })];
let editor: Editor | undefined;

afterEach(() => {
  editor?.destroy();
  editor = undefined;
});

describe("Tiptap v3 editor contracts", () => {
  it("preserves formatting across a JSON content round trip", () => {
    const content = {
      type: "doc",
      content: [
        {
          type: "paragraph",
          content: [{ type: "text", text: "preserved formatting", marks: [{ type: "bold" }] }],
        },
      ],
    };
    editor = new Editor({ extensions, content });

    expect(editor.getJSON()).toEqual(content);
  });

  it("does not emit updates when replacing content with emitUpdate false", () => {
    editor = new Editor({ extensions });
    let updates = 0;
    editor.on("update", () => updates++);

    editor.commands.setContent(
      { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "loaded" }] }] },
      { emitUpdate: false }
    );

    expect(updates).toBe(0);
    expect(editor.getText()).toBe("loaded");
  });

  it("sets the underlying ProseMirror view to non-editable in read-only mode", () => {
    editor = new Editor({ extensions, editable: false });

    expect(editor.isEditable).toBe(false);
    expect(editor.view.editable).toBe(false);
  });
});
