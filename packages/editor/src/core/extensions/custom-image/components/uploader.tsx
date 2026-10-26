/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ImageIcon, RotateCcw } from "lucide-react";
import type { ChangeEvent } from "react";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { useStore } from "@tanstack/react-form";
// plane imports
import { cn } from "@plane/utils/common";
// constants
import { ACCEPTED_IMAGE_MIME_TYPES } from "@/constants/config";
import { CORE_EXTENSIONS } from "@/constants/extension";
// hooks
import {
  canRetryUploadError,
  getUploadError,
  useUploader,
  useDropZone,
  uploadFirstFileAndInsertRemaining,
} from "@/hooks/use-file-upload";
// local imports
import { ECustomImageStatus } from "../types";
import { getImageComponentImageFileMap } from "../utils";
import type { CustomImageNodeViewProps } from "./node-view";

type CustomImageUploaderProps = Pick<
  CustomImageNodeViewProps,
  "editor" | "getPos" | "selected" | "updateAttributes"
> & {
  node: Pick<CustomImageNodeViewProps["node"], "type"> & { attrs: { id: string | null; src: string | null } };
  failedToLoadImage: boolean;
  hasDuplicationFailed: boolean;
  loadImageFromFileSystem: (file: string) => void;
  maxFileSize: number;
  setIsUploaded: (isUploaded: boolean) => void;
};

type TUploadForm = ReturnType<typeof useUploader>;

function getFormErrorMessage(error: unknown) {
  return getUploadError(error)?.message;
}

function UploadAttemptFeedback({
  attempt,
  retryUpload,
}: {
  attempt: TUploadForm["attempts"][number];
  retryUpload: TUploadForm["retryUpload"];
}) {
  const submissionError = useStore(attempt.form.store, (state) => state.errorMap.onSubmit);
  const isSubmitting = useStore(attempt.form.store, (state) => state.isSubmitting);
  const isSubmitSuccessful = useStore(attempt.form.store, (state) => state.isSubmitSuccessful);
  const message = getFormErrorMessage(submissionError);
  if (!message) return null;
  return (
    <div role="alert" className="mt-1 flex items-center gap-2 text-12 text-danger-primary">
      <span>{message}</span>
      {canRetryUploadError(submissionError) && !isSubmitting && !isSubmitSuccessful && (
        <button
          type="button"
          className="underline"
          onClick={(event) => {
            event.stopPropagation();
            void retryUpload(attempt.id);
          }}
        >
          Retry
        </button>
      )}
    </div>
  );
}

function getUploaderClassName({
  selected,
  isEditable,
  isErrorState,
  draggedInside,
}: {
  selected: boolean;
  isEditable: boolean;
  isErrorState: boolean;
  draggedInside: boolean;
}) {
  return cn(
    "image-upload-component flex cursor-default items-center justify-start gap-2 rounded-lg border border-dashed bg-layer-3 px-2 py-3 text-tertiary transition-all duration-200 ease-in-out",
    {
      "border-subtle": !(selected && isEditable && !isErrorState),
      "cursor-pointer hover:bg-layer-3-hover hover:text-secondary": isEditable && !isErrorState,
      "bg-layer-3-hover text-secondary": draggedInside && isEditable && !isErrorState,
      "bg-accent-primary/10 text-accent-secondary hover:bg-accent-primary/10 hover:text-accent-secondary":
        selected && isEditable && !isErrorState,
      "cursor-default bg-danger-subtle text-danger-primary": isErrorState,
      "hover:bg-danger-subtle-hover hover:text-danger-primary": isErrorState && isEditable,
      "bg-danger-subtle-selected": isErrorState && selected,
      "hover:bg-danger-subtle-active": isErrorState && selected && isEditable,
    }
  );
}

function getUploaderMessage({
  isErrorState,
  isUploading,
  draggedInside,
  isEditable,
}: {
  isErrorState: boolean;
  isUploading: boolean;
  draggedInside: boolean;
  isEditable: boolean;
}) {
  if (isErrorState) return "Error loading image";
  if (isUploading) return "Uploading...";
  if (draggedInside && isEditable) return "Drop image here";
  return "Add an image";
}

export function CustomImageUploader(props: CustomImageUploaderProps) {
  const {
    editor,
    failedToLoadImage,
    getPos,
    loadImageFromFileSystem,
    maxFileSize,
    node,
    selected,
    setIsUploaded,
    updateAttributes,
    hasDuplicationFailed,
  } = props;
  // refs
  const uploadId = useRef(Symbol("image upload"));
  const fileInputRef = useRef<HTMLInputElement>(null);
  const hasTriggeredFilePickerRef = useRef(false);
  const hasTriedUploadingOnMountRef = useRef(false);
  const { id: imageEntityId } = node.attrs;
  // derived values
  const imageComponentImageFileMap = useMemo(() => getImageComponentImageFileMap(editor), [editor]);
  const isTouchDevice = !!editor.storage.utility.isTouchDevice;

  const onUpload = useCallback(
    (url: string) => {
      if (url) {
        if (!imageEntityId) return;
        setIsUploaded(true);
        // Update the node view's src attribute post upload
        updateAttributes({
          src: url,
          status: ECustomImageStatus.UPLOADED,
        });
        imageComponentImageFileMap?.delete(imageEntityId);

        const pos = getPos();
        // get current node
        const getCurrentSelection = editor.state.selection;
        const currentNode = editor.state.doc.nodeAt(getCurrentSelection.from);

        // only if the cursor is at the current image component, manipulate
        // the cursor position
        if (
          currentNode &&
          currentNode.type.name === node.type.name &&
          currentNode.attrs.src === url &&
          pos !== undefined
        ) {
          // control cursor position after upload
          const nextNode = editor.state.doc.nodeAt(pos + 1);

          if (nextNode && nextNode.type.name === CORE_EXTENSIONS.PARAGRAPH) {
            // If there is a paragraph node after the image component, move the focus to the next node
            editor.commands.setTextSelection(pos + 1);
          } else {
            // create a new paragraph after the image component post upload
            editor.commands.createParagraphNear();
          }
        }
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [imageComponentImageFileMap, imageEntityId, updateAttributes, getPos]
  );

  const uploadImageEditorCommand = useCallback(
    async (file: File) => {
      const position = getPos();
      const status = position === undefined ? undefined : editor.state.doc.nodeAt(position)?.attrs.status;
      // A failed copy still references somebody else's image. Detach that
      // provisional reference before changing status or starting a replacement.
      const provisional = status === ECustomImageStatus.DUPLICATING || status === ECustomImageStatus.DUPLICATION_FAILED;
      updateAttributes({ ...(provisional ? { src: null } : {}), status: ECustomImageStatus.UPLOADING });
      return await editor.storage.utility.fileHandler.upload(imageEntityId ?? "", file);
    },
    [editor, imageEntityId, updateAttributes, getPos]
  );

  const handleProgressStatus = useCallback(
    (isUploading: boolean) => {
      if (!editor.isDestroyed) editor.commands.setUploadActive(uploadId.current, isUploading);
    },
    [editor]
  );

  useEffect(() => {
    const id = uploadId.current;
    return () => {
      if (!editor.isDestroyed) editor.commands.setUploadActive(id, false);
    };
  }, [editor]);

  // hooks
  const {
    attempts,
    isUploading: isImageBeingUploaded,
    uploadFile,
    retryUpload,
  } = useUploader({
    acceptedMimeTypes: ACCEPTED_IMAGE_MIME_TYPES,
    editorCommand: uploadImageEditorCommand,
    handleProgressStatus,
    loadFileFromFileSystem: loadImageFromFileSystem,
    maxFileSize,
    onUpload,
  });

  const { draggedInside, onDrop, onDragEnter, onDragLeave } = useDropZone({
    editor,
    getPos,
    type: "image",
    uploader: uploadFile,
  });

  // after the image component is mounted we start the upload process based on
  // it's uploaded
  useEffect(() => {
    if (hasTriedUploadingOnMountRef.current) return;

    // the meta data of the image component
    const meta = imageComponentImageFileMap?.get(imageEntityId ?? "");
    if (meta) {
      if (meta.event === "drop" && "file" in meta) {
        hasTriedUploadingOnMountRef.current = true;
        uploadFile(meta.file);
      } else if (meta.event === "insert" && fileInputRef.current && !hasTriggeredFilePickerRef.current) {
        if (meta.hasOpenedFileInputOnce) return;
        if (!isTouchDevice) {
          fileInputRef.current.click();
        }
        hasTriggeredFilePickerRef.current = true;
        imageComponentImageFileMap?.set(imageEntityId ?? "", { ...meta, hasOpenedFileInputOnce: true });
      }
    } else {
      hasTriedUploadingOnMountRef.current = true;
    }
  }, [imageEntityId, isTouchDevice, uploadFile, imageComponentImageFileMap]);

  const onFileChange = useCallback(
    async (e: ChangeEvent<HTMLInputElement>) => {
      e.preventDefault();
      const filesList = e.target.files;
      const pos = getPos();
      if (!filesList || pos === undefined) {
        return;
      }
      await uploadFirstFileAndInsertRemaining({
        editor,
        filesList,
        pos,
        type: "image",
        uploader: uploadFile,
      });
    },
    [uploadFile, editor, getPos]
  );

  const isErrorState = failedToLoadImage || hasDuplicationFailed;

  const borderColor =
    selected && editor.isEditable && !isErrorState
      ? "color-mix(in srgb, var(--border-color-accent-strong) 20%, transparent)"
      : undefined;

  const handleRetryClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      if (hasDuplicationFailed && editor.isEditable) {
        updateAttributes({ status: ECustomImageStatus.DUPLICATING });
      }
    },
    [hasDuplicationFailed, editor.isEditable, updateAttributes]
  );

  return (
    <div
      className={getUploaderClassName({
        selected,
        isEditable: editor.isEditable,
        isErrorState,
        draggedInside,
      })}
      style={borderColor ? { borderColor } : undefined}
      onDrop={onDrop}
      onDragOver={onDragEnter}
      onDragLeave={onDragLeave}
      contentEditable={false}
      onClick={() => {
        if (!failedToLoadImage && editor.isEditable && !hasDuplicationFailed) {
          fileInputRef.current?.click();
        }
      }}
    >
      <ImageIcon className="size-4" />
      <div className="flex-1 text-14 font-medium">
        <div>
          {getUploaderMessage({
            isErrorState,
            isUploading: isImageBeingUploaded,
            draggedInside,
            isEditable: editor.isEditable,
          })}
        </div>
        {attempts.map((attempt) => (
          <UploadAttemptFeedback key={attempt.id} attempt={attempt} retryUpload={retryUpload} />
        ))}
      </div>
      {hasDuplicationFailed && editor.isEditable && (
        <button
          type="button"
          onClick={handleRetryClick}
          className={cn(
            "flex items-center gap-1 rounded-md px-2 py-1 font-medium text-danger-primary transition-all duration-200 ease-in-out hover:bg-danger-subtle-hover",
            {
              "hover:bg-danger-subtle-hover": selected,
            }
          )}
          title="Retry duplication"
        >
          <RotateCcw className="size-3" />
          <span className="text-11">Retry</span>
        </button>
      )}
      <input
        className="size-0 overflow-hidden"
        ref={fileInputRef}
        hidden
        type="file"
        accept={ACCEPTED_IMAGE_MIME_TYPES.join(",")}
        onChange={onFileChange}
        multiple
      />
    </div>
  );
}
