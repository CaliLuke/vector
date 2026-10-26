/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Editor } from "@tiptap/core";
import { Collaboration } from "@tiptap/extension-collaboration";
import { StarterKit } from "@tiptap/starter-kit";
import { expect, it } from "vitest";
import * as Y from "yjs";
import { UniqueID } from "./extension";

it("retains the first text and generated block ID when collaboration supplies an empty paragraph", () => {
  const document = new Y.Doc();
  const fragment = document.getXmlFragment("default");
  fragment.insert(0, [new Y.XmlElement("paragraph")]);
  const editor = new Editor({
    extensions: [
      StarterKit.configure({ undoRedo: false }),
      UniqueID.configure({ types: ["paragraph"] }),
      Collaboration.configure({ document }),
    ],
  });
  try {
    editor.view.dispatch(editor.state.tr.insertText("Saved document", 1));
    expect(editor.getText()).toBe("Saved document");
    expect(editor.state.doc.firstChild?.attrs.id).toEqual(expect.any(String));
    expect(fragment.toJSON()).toContain("Saved document");
  } finally {
    editor.destroy();
    document.destroy();
  }
});
