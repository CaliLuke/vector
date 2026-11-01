import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { act, createRef, useImperativeHandle } from "react";
import type { Ref } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import type { EditorRefApi } from "@/types";
import { getEditorRefHelpers } from "@/helpers/editor-ref";
import { useEditorRef } from "./use-editor-ref";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

function Publisher({ api, ref }: { api: EditorRefApi; ref: Ref<EditorRefApi> }) {
  useImperativeHandle(ref, () => api, [api]);
  return null;
}
function Toolbar({ api, forwardedRef }: { api: EditorRefApi; forwardedRef: Ref<EditorRefApi> }) {
  const { editorRef, ref } = useEditorRef(forwardedRef);
  return (
    <>
      <Publisher api={api} ref={ref} />
      <output>{editorRef?.getDocument().html}</output>
    </>
  );
}

test("toolbar and caller refs receive the current editor API and release replaced or unmounted refs", async () => {
  const editor = new Editor({ extensions: [StarterKit], content: "<p>Draft</p>" });
  const api = getEditorRefHelpers({
    editor,
    provider: undefined,
    getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
  });
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  const objectRef = createRef<EditorRefApi>();
  const callbackRef = vi.fn();
  try {
    await act(async () => root.render(<Toolbar api={api} forwardedRef={objectRef} />));
    expect(objectRef.current).toBe(api);
    expect(host.querySelector("output")?.textContent).toBe("<p>Draft</p>");
    await act(async () => root.render(<Toolbar api={api} forwardedRef={callbackRef} />));
    expect(objectRef.current).toBeNull();
    expect(callbackRef).toHaveBeenLastCalledWith(api);
    await act(async () => root.unmount());
    expect(callbackRef).toHaveBeenLastCalledWith(null);
  } finally {
    editor.destroy();
    host.remove();
  }
});
