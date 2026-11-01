import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { EditorContent } from "@tiptap/react";
import { NodeSelection } from "@tiptap/pm/state";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import type { TFileHandler } from "@/types";
import { CustomImageExtension } from "../extension";
import { UtilityExtension } from "../../utility";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

async function fixture(attributes = 'width="400px" height="200px" aspectRatio="2"') {
  const fileHandler: TFileHandler = {
    assetsUploadStatus: {},
    cancel: () => {},
    checkIfAssetExists: async () => true,
    delete: async () => {},
    restore: async () => {},
    getAssetSrc: async () => "https://example.com/image.png",
    getAssetDownloadSrc: async () => "https://example.com/download",
    upload: async () => "asset",
    duplicate: async () => "copy",
    validation: { maxFileSize: 1000 },
  };
  const editor = new Editor({
    extensions: [
      StarterKit,
      UtilityExtension({
        fileHandler,
        isTouchDevice: false,
        isEditable: true,
        disabledExtensions: [],
        flaggedExtensions: [],
        getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
        mentionHandler: { renderComponent: () => null },
      }).extend({ addProseMirrorPlugins: () => [] }),
      CustomImageExtension({ fileHandler, isEditable: true }),
    ],
    editorProps: { handleScrollToSelection: () => true },
    content: `<image-component id="image" src="asset" status="uploaded" ${attributes}></image-component><p>After</p>`,
  });
  const host = document.createElement("div");
  host.className = "editor-container";
  Object.defineProperty(host, "clientWidth", { value: 1000 });
  document.body.append(host);
  const root = createRoot(host);
  await act(async () => root.render(<EditorContent editor={editor} />));
  const img = host.querySelector("img");
  if (!img) throw new Error("Missing image");
  Object.defineProperties(img, {
    naturalWidth: { value: 800, configurable: true },
    naturalHeight: { value: 400, configurable: true },
  });
  await act(async () => img.dispatchEvent(new Event("load")));
  const resize = host.querySelector<HTMLInputElement>('input[aria-label="Image width"]');
  if (!resize) throw new Error("Missing resize control");
  const bounds = resize.parentElement!;
  vi.spyOn(bounds, "getBoundingClientRect").mockReturnValue(new DOMRect(20, 10, 400, 200));
  return {
    editor,
    host,
    img,
    resize,
    attrs: () => editor.state.doc.firstChild!.attrs,
    update: async (attrs: Record<string, unknown>) =>
      act(async () =>
        editor.view.dispatch(
          editor.state.tr.setNodeMarkup(0, undefined, { ...editor.state.doc.firstChild!.attrs, ...attrs })
        )
      ),
    close: async () => {
      await act(async () => root.unmount());
      editor.destroy();
      host.remove();
    },
  };
}

async function dragStart(resize: HTMLInputElement) {
  await act(async () =>
    resize.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, clientX: 420 }))
  );
}

test("image initialization persists natural aspect ratio and default dimensions once", async () => {
  const f = await fixture();
  try {
    expect(f.attrs()).toMatchObject({ width: "400px", height: "200px", aspectRatio: 2 });
    const transaction = vi.fn();
    f.editor.on("transaction", transaction);
    await act(async () => f.img.dispatchEvent(new Event("load")));
    expect(transaction).not.toHaveBeenCalled();
  } finally {
    await f.close();
  }
  const initial = await fixture();
  try {
    await initial.update({ width: "35%", height: "auto", aspectRatio: null });
    await act(async () => initial.img.dispatchEvent(new Event("load")));
    expect(initial.attrs()).toMatchObject({ width: "350px", height: "175px", aspectRatio: 2 });
  } finally {
    await initial.close();
  }
});

test("image selection and native range changes update the real editor document", async () => {
  const f = await fixture();
  try {
    const select = f.host.querySelector<HTMLButtonElement>('button[aria-label="Select image"]')!;
    await act(async () => {
      select.focus();
      select.click();
    });
    expect(f.editor.state.selection).toBeInstanceOf(NodeSelection);
    expect(f.resize.type).toBe("range");
    expect(f.resize.min).toBe("100");
    await act(async () => {
      f.resize.focus();
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(f.resize, "510");
      f.resize.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(f.attrs()).toMatchObject({ width: "510px", height: "255px" });
    await f.update({ width: "600px", height: "300px" });
    expect(f.resize.value).toBe("600");
    expect(f.img.style.width).toBe("600px");
  } finally {
    await f.close();
  }
});

test("resize saves the latest pointer move even when release occurs in the same event batch", async () => {
  const f = await fixture();
  try {
    await dragStart(f.resize);
    await act(async () => {
      window.dispatchEvent(new MouseEvent("mousemove", { clientX: 620 }));
      window.dispatchEvent(new MouseEvent("mouseup"));
    });
    expect(f.attrs()).toMatchObject({ width: "600px", height: "300px" });
    await f.update({ alignment: "right" });
    await dragStart(f.resize);
    await act(async () => {
      window.dispatchEvent(new MouseEvent("mousemove", { clientX: 400 }));
      window.dispatchEvent(new MouseEvent("mouseup"));
    });
    expect(f.attrs()).toMatchObject({ width: "100px", height: "50px" });
  } finally {
    await f.close();
  }
});

test("remote dimensions cancel an obsolete resize and unmount removes drag listeners", async () => {
  const f = await fixture();
  try {
    await dragStart(f.resize);
    await act(async () => window.dispatchEvent(new MouseEvent("mousemove", { clientX: 700 })));
    await f.update({ width: "300px", height: "150px" });
    await act(async () => window.dispatchEvent(new MouseEvent("mouseup")));
    expect(f.attrs()).toMatchObject({ width: "300px", height: "150px" });
    expect(f.img.style.width).toBe("300px");
    await dragStart(f.resize);
  } finally {
    await f.close();
  }
  expect(() => {
    window.dispatchEvent(new MouseEvent("mousemove", { clientX: 900 }));
    window.dispatchEvent(new MouseEvent("mouseup"));
  }).not.toThrow();
});

test("legacy image dimensions retain their size when the natural aspect ratio is recovered", async () => {
  const f = await fixture('width="420px" height="210px"');
  try {
    expect(f.attrs()).toMatchObject({ width: "420px", height: "210px", aspectRatio: 2 });
  } finally {
    await f.close();
  }
});

test("an image with zero natural height cannot persist invalid dimensions and can load again", async () => {
  const f = await fixture();
  try {
    await f.update({ width: "35%", height: "auto", aspectRatio: null });
    Object.defineProperty(f.img, "naturalHeight", { value: 0, configurable: true });
    await act(async () => f.img.dispatchEvent(new Event("load")));
    expect(f.attrs()).toMatchObject({ width: "35%", height: "auto", aspectRatio: null });
    Object.defineProperty(f.img, "naturalHeight", { value: 400 });
    await act(async () => f.img.dispatchEvent(new Event("load")));
    expect(f.attrs()).toMatchObject({ width: "350px", height: "175px", aspectRatio: 2 });
  } finally {
    await f.close();
  }
});

function touchMove(points: { clientX: number }[]) {
  const event = new Event("touchmove");
  Object.defineProperty(event, "touches", { value: points });
  window.dispatchEvent(event);
}

test("touch cancellation discards the preview, while touch release commits its latest size only once", async () => {
  const f = await fixture();
  try {
    await act(async () => f.resize.dispatchEvent(new Event("touchstart", { bubbles: true, cancelable: true })));
    await act(async () => touchMove([{ clientX: 720 }]));
    expect(f.img.style.width).toBe("700px");
    expect(f.attrs().width).toBe("400px");
    await act(async () => window.dispatchEvent(new Event("touchcancel")));
    expect(f.img.style.width).toBe("400px");
    expect(f.attrs().width).toBe("400px");
    await act(async () => f.resize.dispatchEvent(new Event("touchstart", { bubbles: true, cancelable: true })));
    await act(async () => touchMove([]));
    expect(f.img.style.width).toBe("400px");
    const changed = vi.fn();
    f.editor.on("update", changed);
    await act(async () => {
      touchMove([{ clientX: 620 }]);
      window.dispatchEvent(new Event("touchend"));
      window.dispatchEvent(new Event("touchend"));
      touchMove([{ clientX: 900 }]);
    });
    expect(f.attrs()).toMatchObject({ width: "600px", height: "300px" });
    expect(changed).toHaveBeenCalledOnce();
  } finally {
    await f.close();
  }
});
