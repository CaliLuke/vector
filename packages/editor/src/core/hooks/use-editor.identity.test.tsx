/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { act, useEffect } from "react";
import { createRoot } from "react-dom/client";
import type { Editor } from "@tiptap/core";
import { expect, it, vi } from "vitest";
import { createEditorHookProps } from "../../../tests/helpers/editor-hook-props";
import { useEditor } from "./use-editor";

vi.mock("@/props", () => ({ CoreEditorProps: () => ({ attributes: { class: "base-editor" } }) }));
vi.mock("@/helpers/editor-ref", () => ({ getEditorRefHelpers: () => ({}) }));

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

it("updates real Utility/Markdown/Mention callbacks after commit without replacing editor state", async () => {
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  let current: Editor | null = null;

  function Harness({ displayName }: { displayName: string }) {
    const editor = useEditor(
      createEditorHookProps({
        id: "identity-editor",
        initialValue:
          '<p>Draft <mention-component id="user" entity_identifier="user-id" entity_name="user_mention"></mention-component></p>',
        getEditorMetaData: () => ({
          file_assets: [],
          user_mentions: [{ id: "user-id", display_name: displayName, url: `/people/${displayName}` }],
        }),
        mentionHandler: {
          searchCallback: async () => [],
          renderComponent: () => null,
          getMentionedEntityDetails: () => ({ display_name: displayName }),
        },
      })
    );
    useEffect(() => {
      current = editor;
    }, [editor]);
    return null;
  }

  const render = async (displayName: string) => {
    await act(async () => root.render(<Harness displayName={displayName} />));
  };
  const getEditor = (): Editor => {
    if (!current) throw new Error("Editor did not mount");
    return current;
  };

  try {
    await render("Acme");
    const editor = getEditor();
    const initialInlineContent = editor.getJSON().content?.[0]?.content;
    editor.commands.setTextSelection(5);
    editor.commands.insertContent(" edit");
    const before = editor.getJSON();
    const selection = editor.state.selection.from;
    expect(editor.storage.utility.identityHandlers.mentionHandler.getMentionedEntityDetails?.("user-id")).toEqual({
      display_name: "Acme",
    });
    expect(editor.storage.markdown.getMarkdown()).toContain("@Acme");

    await render("Beta");

    expect(getEditor()).toBe(editor);
    expect(editor.getJSON()).toEqual(before);
    expect(editor.state.selection.from).toBe(selection);
    expect(editor.storage.utility.identityHandlers.getEditorMetaData("<mention-component />").user_mentions).toEqual([
      { id: "user-id", display_name: "Beta", url: "/people/Beta" },
    ]);
    expect(editor.storage.markdown.getMarkdown()).toContain("@Beta");
    expect(editor.can().undo()).toBe(true);
    editor.commands.undo();
    expect(editor.getJSON().content?.[0]?.content).toEqual(initialInlineContent);
    expect(editor.getText()).toBe("Draft @Beta");
  } finally {
    await act(async () => root.unmount());
    host.remove();
  }
});
