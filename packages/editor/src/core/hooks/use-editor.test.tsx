/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { act, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { EditorContent } from "@tiptap/react";
import type { Editor } from "@tiptap/core";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TEditorHookProps } from "@/types";
import { createEditorHookProps } from "../../../tests/helpers/editor-hook-props";
import { useEditor } from "./use-editor";

const editors = vi.hoisted(() => [] as Editor[]);
vi.mock("@/extensions", async () => {
  const { Extension } = await import("@tiptap/core");
  const { default: StarterKit } = await import("@tiptap/starter-kit");
  return {
    CoreEditorExtensions: ({ fileHandler, getEditorMetaData, mentionHandler }: TEditorHookProps) => [
      StarterKit,
      Extension.create({
        name: "utility",
        addStorage: () => {
          return {
            fileHandler,
            identityHandlers: { getEditorMetaData, mentionHandler },
            assetsList: [],
            uploadInProgress: false,
          };
        },
        addCommands: () => ({
          updateAssetsUploadStatus: () => () => true,
          updateIdentityHandlers:
            (identityHandlers) =>
            ({ editor, tr }) => {
              editor.storage.utility.identityHandlers = identityHandlers;
              tr.setMeta("preventDispatch", true);
              return true;
            },
          updateFileHandler:
            (handler) =>
            ({ editor }) => {
              editor.storage.utility.fileHandler = handler;
              return true;
            },
        }),
      }),
    ],
  };
});
vi.mock("@/props", () => ({ CoreEditorProps: () => ({ attributes: { class: "base-editor" } }) }));
vi.mock("@/helpers/editor-ref", () => ({ getEditorRefHelpers: () => ({}) }));
const host = document.createElement("div");
let root: ReturnType<typeof createRoot>;
beforeEach(() => {
  root = createRoot(host);
  editors.length = 0;
});
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
afterEach(async () => {
  await act(async () => {
    root.unmount();
  });
  host.remove();
});

it("skips effects for the destroyed instance when permissions recreate the editor", async () => {
  document.body.append(host);
  let current: Editor | null = null;
  const onEditor = (editor: Editor | null) => {
    current = editor;
  };
  function Harness({ editable }: { editable: boolean }) {
    const editor = useEditor(
      createEditorHookProps({
        id: "test-editor",
        editable,
        initialValue: "<p>Retained content</p>",
        mentionHandler: {
          searchCallback: async () => [],
          renderComponent: () => null,
          getMentionedEntityDetails: () => ({ display_name: "" }),
        },
      })
    );
    useEffect(() => {
      onEditor(editor);
    }, [editor]);
    return <EditorContent editor={editor} />;
  }
  await act(async () => {
    root.render(<Harness editable />);
  });
  const getEditor = (): Editor => {
    if (!current) throw new Error("Editor did not mount");
    return current;
  };
  const initial = getEditor();
  expect(initial).not.toBeNull();
  await act(async () => {
    root.render(<Harness editable={false} />);
  });
  expect(initial.isDestroyed).toBe(true);
  expect(getEditor().isEditable).toBe(false);
  expect(getEditor().getText()).toBe("Retained content");
  await act(async () => {
    root.render(<Harness editable />);
  });
  expect(getEditor().isEditable).toBe(true);
});

it("editor storage receives the current file owner without recreating the editor or losing its draft", async () => {
  document.body.append(host);
  const firstUpload = vi.fn(async () => "first-asset");
  const nextUpload = vi.fn(async () => "next-asset");
  const firstDuplicate = vi.fn(async () => "first-copy");
  const nextDuplicate = vi.fn(async () => "next-copy");
  function Harness({ next }: { next: boolean }) {
    const editor = useEditor(
      createEditorHookProps({
        id: "draft-editor",
        editorProps: { attributes: () => ({ "aria-label": next ? "Updated notes" : "Notes" }) },
        initialValue: "<p>Unsaved draft</p>",
        fileHandler: {
          assetsUploadStatus: {},
          cancel: () => {},
          checkIfAssetExists: async () => true,
          delete: async () => {},
          restore: async () => {},
          getAssetSrc: async (path) => path,
          getAssetDownloadSrc: async (path) => path,
          upload: next ? nextUpload : firstUpload,
          duplicate: next ? nextDuplicate : firstDuplicate,
          validation: { maxFileSize: 1000 },
        },
        mentionHandler: {
          searchCallback: async () => [],
          renderComponent: () => null,
          getMentionedEntityDetails: () => ({ display_name: "" }),
        },
      })
    );
    useEffect(() => {
      if (editor) editors.push(editor);
    }, [editor]);
    return <EditorContent editor={editor} />;
  }
  await act(async () => {
    root.render(<Harness next={false} />);
  });
  const installed = editors.at(-1);
  if (!installed) throw new Error("Editor did not install its extensions");
  await act(async () => {
    root.render(<Harness next />);
  });
  const file = new File(["image"], "image.png");
  expect(await installed.storage.utility.fileHandler.upload("block", file)).toBe("next-asset");
  expect(await installed.storage.utility.fileHandler.duplicate("original")).toBe("next-copy");
  expect(firstUpload).not.toHaveBeenCalled();
  expect(firstDuplicate).not.toHaveBeenCalled();
  expect(nextUpload).toHaveBeenCalledWith("block", file);
  expect(nextDuplicate).toHaveBeenCalledWith("original");
  expect(host.textContent).toContain("Unsaved draft");
  const content = host.querySelector(".tiptap");
  expect(content?.classList.contains("base-editor")).toBe(true);
  expect(content?.getAttribute("aria-label")).toBe("Updated notes");
});

it("commits current mention handlers without replacing the editor or its selection", async () => {
  document.body.append(host);
  const installedEditors: Editor[] = [];
  function Harness({ workspace }: { workspace: "acme" | "beta" | "missing" }) {
    const editor = useEditor(
      createEditorHookProps({
        id: "identity-editor",
        initialValue: "@user",
        getEditorMetaData: () => ({
          file_assets: [],
          user_mentions: [{ id: "user", display_name: workspace, url: `/${workspace}/profile/user` }],
        }),
        mentionHandler: {
          searchCallback: async () => [],
          renderComponent: () => null,
          getMentionedEntityDetails: () => (workspace === "missing" ? undefined : { display_name: workspace }),
        },
      })
    );
    useEffect(() => {
      if (editor && !installedEditors.includes(editor)) installedEditors.push(editor);
    }, [editor]);
    return <EditorContent editor={editor} />;
  }

  await act(async () => {
    root.render(<Harness workspace="acme" />);
  });
  const installed = installedEditors.at(-1);
  if (!installed) throw new Error("Editor identity extensions did not mount");
  installed.commands.setTextSelection(3);

  await act(async () => {
    root.render(<Harness workspace="beta" />);
  });

  expect(installedEditors.at(-1)).toBe(installed);
  expect(installed.state.selection.from).toBe(3);
  expect(installed.storage.utility.identityHandlers.mentionHandler.getMentionedEntityDetails?.("user")).toEqual({
    display_name: "beta",
  });
  expect(installed.storage.utility.identityHandlers.getEditorMetaData("<mention-component />").user_mentions).toEqual([
    { id: "user", display_name: "beta", url: "/beta/profile/user" },
  ]);

  await act(async () => {
    root.render(<Harness workspace="missing" />);
  });
  expect(
    installed.storage.utility.identityHandlers.mentionHandler.getMentionedEntityDetails?.("missing-user")
  ).toBeUndefined();
  expect(installedEditors.at(-1)).toBe(installed);
  expect(installed.state.selection.from).toBe(3);
});
