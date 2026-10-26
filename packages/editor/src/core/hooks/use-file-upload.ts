/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor, NodeViewProps } from "@tiptap/core";
import type { DragEvent } from "react";
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import { FormApi } from "@tanstack/react-form";
// helpers
import type { EFileError } from "@/helpers/file";
import { isFileValid } from "@/helpers/file";
// plugins
import { insertFilesSafely } from "@/plugins/drop";
// types
import type { TEditorCommands } from "@/types";

type TUploaderArgs = {
  acceptedMimeTypes: string[];
  editorCommand: (file: File) => Promise<string | undefined>;
  handleProgressStatus?: (isUploading: boolean) => void;
  loadFileFromFileSystem?: (file: string) => void;
  maxFileSize: number;
  onUpload: (url: string, file: File) => void;
};

export type TUploadError = { kind: "validation" | "transport" | "accepted"; message: string };

export function getUploadError(error: unknown): TUploadError | undefined {
  if (
    error &&
    typeof error === "object" &&
    "kind" in error &&
    (error.kind === "validation" || error.kind === "transport" || error.kind === "accepted") &&
    "message" in error &&
    typeof error.message === "string"
  ) {
    return { kind: error.kind, message: error.message };
  }
  return undefined;
}

export const canRetryUploadError = (error: unknown) => getUploadError(error)?.kind === "transport";

const uploadErrorMessage = (error: unknown) =>
  error instanceof Error && error.message ? error.message : "Something went wrong while uploading the file.";

function getFileValidationError(
  file: File,
  acceptedMimeTypes: string[],
  maxFileSize: number
): TUploadError | undefined {
  let validationError: { code: EFileError; message: string } | undefined;
  const isValid = isFileValid({
    acceptedMimeTypes,
    file,
    maxFileSize,
    onError: (code, message) => {
      validationError = { code, message };
    },
  });
  return isValid
    ? undefined
    : { kind: "validation" as const, message: validationError?.message ?? "This file cannot be uploaded." };
}

function createUploadForm(args: TUploaderArgs & { file: File }) {
  const {
    acceptedMimeTypes,
    editorCommand,
    file,
    handleProgressStatus,
    loadFileFromFileSystem,
    maxFileSize,
    onUpload,
  } = args;
  return new FormApi({
    defaultValues: { file },
    validators: {
      onSubmit: ({ value }) => getFileValidationError(value.file, acceptedMimeTypes, maxFileSize),
    },
    onSubmit: async ({ value, formApi }) => {
      handleProgressStatus?.(true);
      let uploadAccepted = false;
      let previewReadFailed = false;
      const reportPreviewFailure = () => {
        if (!uploadAccepted) {
          previewReadFailed = true;
          return;
        }
        formApi.setErrorMap({
          onSubmit: { kind: "accepted", message: "The image preview could not be read." },
        });
      };
      if (loadFileFromFileSystem) {
        const reader = new FileReader();
        reader.addEventListener("load", () => {
          if (typeof reader.result === "string") loadFileFromFileSystem(reader.result);
          else reportPreviewFailure();
        });
        reader.addEventListener("error", reportPreviewFailure);
        reader.readAsDataURL(value.file);
      }

      let url: string;
      try {
        url = (await editorCommand(value.file)) ?? "";
        if (!url) throw new Error("Something went wrong while uploading the file.");
      } catch (error: unknown) {
        formApi.setErrorMap({
          onSubmit: { kind: "transport", message: uploadErrorMessage(error) },
        });
        throw error;
      }
      uploadAccepted = true;
      if (previewReadFailed) reportPreviewFailure();

      try {
        onUpload(url, value.file);
      } catch {
        formApi.setErrorMap({
          onSubmit: {
            kind: "accepted",
            message: "The upload completed, but the image could not be inserted into the editor.",
          },
        });
      }
    },
  });
}

type TUploadForm = ReturnType<typeof createUploadForm>;
type TUploadAttempt = { id: number; form: TUploadForm; cleanup: () => void };

export const useUploader = (args: TUploaderArgs) => {
  const { acceptedMimeTypes, editorCommand, handleProgressStatus, loadFileFromFileSystem, maxFileSize, onUpload } =
    args;
  const attemptSequence = useRef(0);
  const attemptsRef = useRef<TUploadAttempt[]>([]);
  const [attempts, setAttempts] = useState<TUploadAttempt[]>([]);

  useEffect(
    () => () => {
      attemptsRef.current.forEach((attempt) => attempt.cleanup());
      attemptsRef.current = [];
    },
    []
  );

  const subscribe = useCallback(
    (onStoreChange: () => void) => {
      const unsubscribers = attempts.map((attempt) => attempt.form.store.subscribe(onStoreChange));
      return () => unsubscribers.forEach((subscription) => subscription.unsubscribe());
    },
    [attempts]
  );
  const getIsUploading = useCallback(() => attempts.some((attempt) => attempt.form.state.isSubmitting), [attempts]);
  const isUploading = useSyncExternalStore(subscribe, getIsUploading, () => false);

  const submitAttempt = useCallback(
    async (attempt: TUploadAttempt) => {
      await attempt.form.handleSubmit().catch(() => undefined);
      if (!attemptsRef.current.some((current) => current.form.state.isSubmitting)) {
        handleProgressStatus?.(false);
      }
    },
    [handleProgressStatus]
  );

  const createAttempt = useCallback(
    (file: File): TUploadAttempt => {
      const form = createUploadForm({
        acceptedMimeTypes,
        editorCommand,
        handleProgressStatus,
        loadFileFromFileSystem,
        maxFileSize,
        onUpload,
        file,
      });
      return { id: attemptSequence.current++, form, cleanup: form.mount() };
    },
    [acceptedMimeTypes, editorCommand, handleProgressStatus, loadFileFromFileSystem, maxFileSize, onUpload]
  );

  const uploadFile = useCallback(
    async (file: File) => {
      const attempt = createAttempt(file);
      const nextAttempts = [...attemptsRef.current, attempt];
      attemptsRef.current = nextAttempts;
      setAttempts(nextAttempts);
      await submitAttempt(attempt);
    },
    [createAttempt, submitAttempt]
  );

  const retryUpload = useCallback(
    async (attemptId: number) => {
      const failed = attemptsRef.current.find(({ id }) => id === attemptId);
      if (
        !failed ||
        failed.form.state.isSubmitting ||
        failed.form.state.isSubmitSuccessful ||
        !canRetryUploadError(failed.form.state.errorMap.onSubmit)
      ) {
        return;
      }
      failed.cleanup();
      const retry = createAttempt(failed.form.state.values.file);
      const nextAttempts = attemptsRef.current.map((attempt) => (attempt.id === attemptId ? retry : attempt));
      attemptsRef.current = nextAttempts;
      setAttempts(nextAttempts);
      await submitAttempt(retry);
    },
    [createAttempt, submitAttempt]
  );

  return { attempts, isUploading, uploadFile, retryUpload };
};

type TDropzoneArgs = {
  editor: Editor;
  getPos: NodeViewProps["getPos"];
  type: Extract<TEditorCommands, "attachment" | "image">;
  uploader: (file: File) => Promise<void>;
};

export const useDropZone = (args: TDropzoneArgs) => {
  const { editor, getPos, type, uploader } = args;
  // states
  const [isDragging, setIsDragging] = useState<boolean>(false);
  const [draggedInside, setDraggedInside] = useState<boolean>(false);

  useEffect(() => {
    const dragStartHandler = () => {
      setIsDragging(true);
    };

    const dragEndHandler = () => {
      setIsDragging(false);
    };

    document.body.addEventListener("dragstart", dragStartHandler);
    document.body.addEventListener("dragend", dragEndHandler);

    return () => {
      document.body.removeEventListener("dragstart", dragStartHandler);
      document.body.removeEventListener("dragend", dragEndHandler);
    };
  }, []);

  const onDrop = useCallback(
    async (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setDraggedInside(false);
      const filesList = e.dataTransfer.files;
      const pos = getPos();

      if (filesList.length === 0 || !editor.isEditable || pos === undefined) {
        return;
      }

      await uploadFirstFileAndInsertRemaining({
        editor,
        filesList,
        pos,
        type,
        uploader,
      });
    },
    [editor, type, uploader, getPos]
  );
  const onDragEnter = useCallback(() => setDraggedInside(true), []);
  const onDragLeave = useCallback(() => setDraggedInside(false), []);

  return {
    isDragging,
    draggedInside,
    onDragEnter,
    onDragLeave,
    onDrop,
  };
};

type TMultipleFileArgs = {
  editor: Editor;
  filesList: FileList;
  pos: number;
  type: Extract<TEditorCommands, "attachment" | "image">;
  uploader: (file: File) => Promise<void>;
};

// Upload the first file and insert the remaining ones for uploading multiple files
export const uploadFirstFileAndInsertRemaining = async (args: TMultipleFileArgs) => {
  const { editor, filesList, pos, type, uploader } = args;
  const filesArray = Array.from(filesList);
  if (filesArray.length === 0) {
    console.error("No files found to upload.");
    return;
  }

  // Upload the first file
  const firstFile = filesArray[0];
  uploader(firstFile);
  // Insert the remaining files
  const remainingFiles = filesArray.slice(1);
  if (remainingFiles.length > 0) {
    const docSize = editor.state.doc.content.size;
    const posOfNextFileToBeInserted = Math.min(pos + 1, docSize);
    insertFilesSafely({
      editor,
      files: remainingFiles,
      initialPos: posOfNextFileToBeInserted,
      event: "drop",
      type,
    });
  }
};
