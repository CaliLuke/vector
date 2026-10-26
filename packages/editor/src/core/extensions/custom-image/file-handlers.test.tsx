/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { act } from "react";
import { createRoot } from "react-dom/client";
import type { ComponentProps } from "react";
import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { EditorContent } from "@tiptap/react";
import { expect, it, vi } from "vitest";
import type { TFileHandler } from "@/types";
import { UtilityExtension } from "../utility";
import { CustomImageExtension } from "./extension";
import { TrackFileDeletionPlugin } from "../../plugins/file/delete";
import { getEditorRefHelpers } from "../../helpers/editor-ref";
import { processAssetDuplication } from "../../helpers/paste-asset";
import { CustomImageUploader } from "./components/uploader";

type TAttributeUpdater = ComponentProps<typeof CustomImageUploader>["updateAttributes"];

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((done, fail) => {
    resolve = done;
    reject = fail;
  });
  return { promise, resolve, reject };
}

function createUploadEditor(upload: TFileHandler["upload"]) {
  const fileHandler: TFileHandler = {
    assetsUploadStatus: {},
    cancel: () => {},
    checkIfAssetExists: async () => true,
    delete: async () => {},
    restore: async () => {},
    getAssetSrc: async (path) => path,
    getAssetDownloadSrc: async (path) => path,
    upload,
    duplicate: async (id) => id,
    validation: { maxFileSize: 1000 },
  };
  const imageExtension = CustomImageExtension({ fileHandler, isEditable: true });
  const editor = new Editor({
    extensions: [
      StarterKit,
      UtilityExtension({
        fileHandler,
        disabledExtensions: [],
        flaggedExtensions: [],
        getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
        mentionHandler: { renderComponent: () => null },
        isEditable: true,
        isTouchDevice: false,
      }).extend({
        // This fixture exercises storage commands and upload UI, independently of clipboard/drop plugins.
        addProseMirrorPlugins: () => [],
      }),
      imageExtension,
    ],
    content: "<p>Draft</p>",
  });
  return { editor, imageExtension, fileHandler };
}

async function mountImageUploader(upload: TFileHandler["upload"], updateAttributes: TAttributeUpdater) {
  const { editor, imageExtension } = createUploadEditor(upload);
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  await act(async () => {
    root.render(
      <CustomImageUploader
        editor={editor}
        node={{ type: editor.schema.nodes[imageExtension.name], attrs: { id: "block", src: null } }}
        getPos={() => 0}
        selected={false}
        updateAttributes={updateAttributes}
        failedToLoadImage={false}
        hasDuplicationFailed={false}
        loadImageFromFileSystem={() => {}}
        maxFileSize={1000}
        setIsUploaded={() => {}}
      />
    );
  });
  return {
    editor,
    host,
    unmount: async () => {
      await act(async () => root.unmount());
      editor.destroy();
      host.remove();
    },
  };
}

async function selectImageFile(host: HTMLElement) {
  const input = host.querySelector<HTMLInputElement>('input[type="file"]');
  if (!input) throw new Error("File input was not rendered");
  const file = new File(["image"], "image.png", { type: "image/png" });
  Object.defineProperty(input, "files", { value: [file] });
  await act(async () => input.dispatchEvent(new Event("change", { bubbles: true })));
  return file;
}

it("a mounted image uploader uses the current asset owner without losing the editor draft", async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const originalUpload = vi.fn(async () => "original-asset");
  const currentUpload = vi.fn(async () => "current-asset");
  const { editor, imageExtension, fileHandler } = createUploadEditor(originalUpload);
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  const updateAttributes = vi.fn();
  const props = {
    editor,
    extension: imageExtension,
    node: { type: editor.schema.nodes[imageExtension.name], attrs: { id: "block", src: null } },
    getPos: () => 0,
    selected: false,
    updateAttributes,
    deleteNode: () => {},
    decorations: [],
    innerDecorations: [],
    HTMLAttributes: {},
    view: editor.view,
  };
  try {
    await act(async () => {
      root.render(
        <CustomImageUploader
          {...props}
          failedToLoadImage={false}
          hasDuplicationFailed={false}
          loadImageFromFileSystem={() => {}}
          maxFileSize={1000}
          setIsUploaded={() => {}}
        />
      );
    });
    editor.commands.updateFileHandler({ ...fileHandler, upload: currentUpload });
    const input = host.querySelector('input[type="file"]');
    if (!input) throw new Error("File input was not rendered");
    const file = new File(["image"], "image.png", { type: "image/png" });
    Object.defineProperty(input, "files", { value: [file] });
    await act(async () => {
      input.dispatchEvent(new Event("change", { bubbles: true }));
    });
    expect(originalUpload).not.toHaveBeenCalled();
    expect(currentUpload).toHaveBeenCalledWith("block", file);
    expect(updateAttributes).toHaveBeenCalledWith(expect.objectContaining({ src: "current-asset" }));
    expect(editor.getText()).toBe("Draft");
  } finally {
    await act(async () => root.unmount());
    editor.destroy();
    host.remove();
  }
});

it("keeps a failed upload visible and retries only that unaccepted file", async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const upload = vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce("saved-asset");
  const updateAttributes = vi.fn();
  const mounted = await mountImageUploader(upload, updateAttributes);
  const { host } = mounted;
  try {
    const file = await selectImageFile(host);

    expect(host.querySelector('[role="alert"]')?.textContent).toContain("offline");
    expect(upload).toHaveBeenCalledTimes(1);
    const retry = [...host.querySelectorAll("button")].find((button) => button.textContent === "Retry");
    if (!retry) throw new Error("Failed upload did not expose a retry action");
    await act(async () => retry.click());

    expect(upload).toHaveBeenCalledTimes(2);
    expect(upload).toHaveBeenLastCalledWith("block", file);
    expect(updateAttributes).toHaveBeenCalledWith(expect.objectContaining({ src: "saved-asset" }));
    expect(host.querySelector('[role="alert"]')).toBeNull();
  } finally {
    await mounted.unmount();
  }
});

it("does not retry an upload that the server accepted when editor insertion fails", async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const upload = vi.fn(async () => "saved-asset");
  const updateAttributes = vi.fn((attributes: object) => {
    if ("src" in attributes) throw new Error("editor transaction failed");
  });
  const mounted = await mountImageUploader(upload, updateAttributes);
  const { host } = mounted;
  try {
    const file = await selectImageFile(host);

    expect(upload).toHaveBeenCalledExactlyOnceWith("block", file);
    expect(host.querySelector('[role="alert"]')?.textContent).toContain("upload completed");
    expect([...host.querySelectorAll("button")].some((button) => button.textContent === "Retry")).toBe(false);
  } finally {
    await mounted.unmount();
  }
});

it("one completed image cannot make an editor discardable while another upload is pending", () => {
  const { editor } = createUploadEditor(vi.fn(async () => "asset"));
  const first = Symbol("first");
  const second = Symbol("second");
  try {
    editor.commands.setUploadActive(first, true);
    editor.commands.setUploadActive(second, true);
    editor.commands.setUploadActive(first, false);
    expect(editor.storage.utility.uploadInProgress).toBe(true);
    editor.commands.setUploadActive(second, false);
    expect(editor.storage.utility.uploadInProgress).toBe(false);
  } finally {
    editor.destroy();
  }
});

it.each(["duplicating", "duplication-failed"])(
  "replacing a %s image never deletes its source asset",
  async (status) => {
    const { editor } = createUploadEditor(vi.fn(async () => "unused"));
    const remove = vi.fn(async () => {});
    try {
      editor.commands.setContent(`<image-component id="copy" src="original" status="${status}"></image-component>`);
      editor.registerPlugin(TrackFileDeletionPlugin(editor, remove));
      editor.view.dispatch(
        editor.state.tr.setNodeMarkup(0, undefined, {
          ...editor.state.doc.firstChild!.attrs,
          src: "independent-copy",
          status: "uploaded",
        })
      );
      expect(remove).not.toHaveBeenCalled();
      editor.commands.deleteRange({ from: 0, to: editor.state.doc.firstChild!.nodeSize });
      expect(remove).toHaveBeenCalledExactlyOnceWith("independent-copy");
    } finally {
      editor.destroy();
    }
  }
);

it("copied images get separate block IDs, including legacy private images", () => {
  const asset = "11111111-1111-4111-8111-111111111111";
  const html = `<image-component src='${asset}'></image-component><image-component src='${asset}'></image-component><img src='${asset}'><img src='https://example.com/public.png'>`;
  const { processedHtml } = processAssetDuplication(html);
  const container = document.createElement("div");
  container.innerHTML = processedHtml;
  const images = [...container.querySelectorAll("image-component")];
  expect(images).toHaveLength(3);
  expect(new Set(images.map((image) => image.id)).size).toBe(3);
  expect(images.every((image) => image.getAttribute("status") === "duplicating")).toBe(true);
  expect(container.querySelector("img")?.getAttribute("src")).toBe("https://example.com/public.png");
});

it.each([false, true])("replacing a failed copy protects its source when upload failure is %s", async (fails) => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  let finish!: (value: string) => void;
  let reject!: (error: Error) => void;
  const pending = new Promise<string>((resolve, fail) => {
    finish = resolve;
    reject = fail;
  });
  const { editor } = createUploadEditor(() => pending);
  const remove = vi.fn(async () => {});
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  const ref = getEditorRefHelpers({
    editor,
    provider: undefined,
    getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
  });
  try {
    editor.commands.setContent(
      '<image-component id="copy" src="original" status="duplication-failed"></image-component><p>Draft</p>'
    );
    editor.registerPlugin(TrackFileDeletionPlugin(editor, remove));
    await act(async () =>
      root.render(
        <CustomImageUploader
          editor={editor}
          node={{ type: editor.state.doc.firstChild!.type, attrs: { id: "copy", src: "original" } }}
          getPos={() => 0}
          selected={false}
          updateAttributes={(attrs) =>
            editor.view.dispatch(
              editor.state.tr.setNodeMarkup(0, undefined, { ...editor.state.doc.firstChild!.attrs, ...attrs })
            )
          }
          failedToLoadImage={false}
          hasDuplicationFailed
          loadImageFromFileSystem={() => {}}
          maxFileSize={1000}
          setIsUploaded={() => {}}
        />
      )
    );
    const input = host.querySelector('input[type="file"]')!;
    Object.defineProperty(input, "files", { value: [new File(["image"], "image.png", { type: "image/png" })] });
    await act(async () => input.dispatchEvent(new Event("change", { bubbles: true })));
    expect(editor.state.doc.firstChild!.attrs.src).toBeNull();
    expect(ref.isEditorReadyToDiscard()).toBe(false);
    expect(remove).not.toHaveBeenCalled();
    await act(async () => {
      if (fails) reject(new Error("offline"));
      else finish("replacement");
    });
    expect(remove).not.toHaveBeenCalled();
    expect(ref.isEditorReadyToDiscard()).toBe(!fails);
    editor.commands.deleteRange({ from: 0, to: editor.state.doc.firstChild!.nodeSize });
    expect(remove.mock.calls).toEqual(fails ? [] : [["replacement"]]);
    expect(ref.isEditorReadyToDiscard()).toBe(true);
  } finally {
    await act(async () => root.unmount());
    editor.destroy();
    host.remove();
  }
});

it.each(["duplicating", "duplication-failed"])("a %s image blocks submission until copied or removed", (status) => {
  const { editor } = createUploadEditor(vi.fn(async () => "unused"));
  const ref = getEditorRefHelpers({
    editor,
    provider: undefined,
    getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
  });
  try {
    editor.commands.setContent(`<image-component src="source" status="${status}"></image-component>`);
    expect(ref.isEditorReadyToDiscard()).toBe(false);
    editor.commands.deleteRange({ from: 0, to: editor.state.doc.firstChild!.nodeSize });
    expect(ref.isEditorReadyToDiscard()).toBe(true);
  } finally {
    editor.destroy();
  }
});

it.each(["before", "after", "failure"])(
  "a pending copy resolving %s replacement cannot overwrite the replacement",
  async (order) => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    const copied = deferred<string>();
    const uploaded = deferred<string>();
    const { editor, fileHandler } = createUploadEditor(() => uploaded.promise);
    editor.commands.updateFileHandler({
      ...fileHandler,
      duplicate: () => copied.promise,
      getAssetSrc: async () => "",
    });
    editor.commands.setContent(
      '<image-component id="copy" src="source" status="duplicating"></image-component><p>Draft</p>'
    );
    const host = document.createElement("div");
    document.body.append(host);
    const root = createRoot(host);
    try {
      await act(async () => root.render(<EditorContent editor={editor} />));
      const input = host.querySelector('input[type="file"]');
      expect(input).not.toBeNull();
      Object.defineProperty(input!, "files", { value: [new File(["image"], "image.png", { type: "image/png" })] });
      await act(async () => input!.dispatchEvent(new Event("change", { bubbles: true })));
      expect(editor.state.doc.firstChild!.attrs.src).toBeNull();
      if (order === "before") await act(async () => copied.resolve("late-copy"));
      await act(async () => uploaded.resolve("replacement"));
      if (order === "after") await act(async () => copied.resolve("late-copy"));
      if (order === "failure") await act(async () => copied.reject(new Error("copy failed")));
      expect(editor.state.doc.firstChild!.attrs.src).toBe("replacement");
      expect(editor.state.doc.firstChild!.attrs.status).toBe("uploaded");
    } finally {
      await act(async () => root.unmount());
      editor.destroy();
      host.remove();
    }
  }
);
