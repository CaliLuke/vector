import { useCallback, useEffect, useLayoutEffect, useRef } from "react";
import type { RefObject } from "react";
import type { EditorRefApi } from "@plane/editor";
import { EFileAssetType } from "@plane/types";
import { RichTextEditor } from "@/components/editor/rich-text";
import { WorkspaceService } from "@/services/workspace.service";
const workspaceService = new WorkspaceService();

type Props = {
  id: string;
  propertyId: string;
  value: string;
  workspaceSlug: string;
  workspaceId: string;
  projectId?: string;
  itemId?: string;
  isDraft: boolean;
  onChange: (html: string) => void;
  onAssetUpload: (id: string) => void;
  editors: Map<string, EditorRefApi>;
  onEditorStateChange: () => void;
  attributes: Record<string, string>;
};
/** Apply accepted external values without feeding each local keystroke back through setContent. */
export function WorkItemRichTextField(props: Props) {
  const {
    id,
    propertyId,
    value,
    workspaceSlug,
    workspaceId,
    projectId,
    itemId,
    isDraft,
    onChange,
    onAssetUpload,
    editors,
    onEditorStateChange,
  } = props;
  const editor = useRef<EditorRefApi | null>(null);
  const rendered = useRef(value);
  const unsubscribe = useRef<(() => void) | undefined>(undefined);
  const stateChanged = useRef(onEditorStateChange);
  const external = useRef(value);
  useLayoutEffect(() => {
    stateChanged.current = onEditorStateChange;
    external.current = value;
  }, [onEditorStateChange, value]);
  const register = useCallback(
    (instance: EditorRefApi | null) => {
      unsubscribe.current?.();
      unsubscribe.current = undefined;
      editor.current = instance;
      if (instance) editors.set(propertyId, instance);
      else editors.delete(propertyId);
      if (instance) {
        const handleStateChange = () => {
          // Wait for the transaction's onChange to record local edits before
          // accepting an external value deferred during an upload.
          queueMicrotask(() => {
            if (editor.current !== instance) return;
            stateChanged.current();
            synchronizeEditorValue(instance, rendered, external.current);
          });
        };
        unsubscribe.current = instance.onStateChange(handleStateChange);
        // A replacement handle must retry deferred hydration even when its
        // predecessor's queued callback was discarded.
        handleStateChange();
      }
      synchronizeEditorValue(instance, rendered, external.current);
    },
    [editors, propertyId]
  );
  useEffect(() => {
    synchronizeEditorValue(editor.current, rendered, value);
  }, [value]);
  return (
    <RichTextEditor
      id={id}
      // The parent fieldset locks submissions with inert; changing editable
      // would recreate TipTap and discard undo history on a rejected save.
      editable
      editorProps={{ attributes: props.attributes }}
      initialValue={value}
      value={null}
      ref={register}
      workspaceSlug={workspaceSlug}
      workspaceId={workspaceId}
      projectId={projectId}
      onChange={(_json, html) => {
        rendered.current = html;
        external.current = html;
        onChange(html);
      }}
      containerClassName="min-h-28 pt-3"
      fileAsset={{
        entity_identifier: itemId ?? "",
        entity_type: isDraft ? EFileAssetType.DRAFT_ISSUE_DESCRIPTION : EFileAssetType.ISSUE_DESCRIPTION,
      }}
      onAssetUpload={onAssetUpload}
      searchMentionCallback={(payload) =>
        workspaceService.searchEntity(workspaceSlug, { ...payload, ...(projectId ? { project_id: projectId } : {}) })
      }
    />
  );
}
function synchronizeEditorValue(editor: EditorRefApi | null, rendered: RefObject<string>, value: string) {
  if (!editor || rendered.current === value || !editor.isEditorReadyToDiscard()) return;
  rendered.current = value;
  editor.setEditorValue(value, false);
}
