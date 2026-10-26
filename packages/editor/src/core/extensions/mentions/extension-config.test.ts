/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Editor, Extension, Node } from "@tiptap/core";
import { Markdown } from "tiptap-markdown";
import { afterEach, expect, it } from "vitest";
import { CustomMentionExtension } from "./extension";

const documentExtension = Node.create({ name: "doc", topNode: true, content: "block+" });
const paragraphExtension = Node.create({
  name: "paragraph",
  group: "block",
  content: "inline*",
  parseHTML: () => [{ tag: "p" }],
  renderHTML: () => ["p", 0],
});
const textExtension = Node.create({ name: "text", group: "inline" });
const identityStorageExtension = Extension.create({
  name: "utility",
  addStorage() {
    return {
      identityHandlers: {
        getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
        mentionHandler: {
          searchCallback: async () => [],
          renderComponent: () => null,
          getMentionedEntityDetails: () => ({ display_name: "Acme" }),
        },
      },
    };
  },
});

let editor: Editor | undefined;
afterEach(() => {
  editor?.destroy();
  editor = undefined;
});

it("serializes mention details from committed identity storage without replacing the editor document", async () => {
  const host = document.createElement("div");
  document.body.append(host);
  editor = new Editor({
    element: host,
    content:
      '<p><mention-component id="user" entity_identifier="member-id" entity_name="user_mention"></mention-component></p>',
    extensions: [
      documentExtension,
      paragraphExtension,
      textExtension,
      Markdown.configure({ html: true, transformCopiedText: false, transformPastedText: true, breaks: true }),
      identityStorageExtension,
      CustomMentionExtension(),
    ],
  });

  const mentionEditor = editor;
  await new Promise<void>((resolve) => mentionEditor.on("create", () => resolve()));
  expect(mentionEditor.getText()).toBe("@Acme");
  expect(mentionEditor.storage.markdown.getMarkdown()).toContain("@Acme");
  mentionEditor.commands.setTextSelection(2);
  const documentBeforeUpdate = mentionEditor.state.doc;
  const selectionBeforeUpdate = mentionEditor.state.selection.from;
  const handler = mentionEditor.storage.utility.identityHandlers.mentionHandler;
  mentionEditor.storage.utility.identityHandlers.mentionHandler = {
    ...handler,
    getMentionedEntityDetails: () => ({ display_name: "Beta" }),
  };

  expect(mentionEditor.getText()).toBe("@Beta");
  expect(mentionEditor.storage.markdown.getMarkdown()).toContain("@Beta");
  expect(mentionEditor.state.doc).toBe(documentBeforeUpdate);
  expect(mentionEditor.state.selection.from).toBe(selectionBeforeUpdate);
  mentionEditor.storage.utility.identityHandlers.mentionHandler = {
    ...handler,
    getMentionedEntityDetails: () => undefined,
  };
  expect(mentionEditor.getText()).toBe("@user");
  expect(mentionEditor.storage.markdown.getMarkdown()).toContain("@user");
  host.remove();
});
