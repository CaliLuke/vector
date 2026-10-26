/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect, useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { useDropzone } from "react-dropzone";
import { ACCEPTED_AVATAR_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE, MAX_FILE_SIZE } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { UserCirclePropertyIcon } from "@plane/propel/icons";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
import { ImagePreview } from "@/components/core/image-preview";

export type ImageUploadModalProps = {
  onClose: () => void;
  onSuccess: (url: string) => void | Promise<void>;
  handleRemove: () => void | Promise<void>;
  value: string | null;
};

type Props = ImageUploadModalProps & {
  upload: (file: File, signal: AbortSignal) => Promise<string>;
  remove: (url: string) => Promise<void>;
};

export function ImageUploadModal({ onClose, onSuccess, handleRemove, value, upload, remove }: Props) {
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  const uploadMutation = useMutation({
    retry: false,
    mutationFn: async (file: File) => {
      const controller = new AbortController();
      request.current = controller;
      const url = await upload(file, controller.signal);
      controller.signal.throwIfAborted();
      return { file, url };
    },
  });
  const removeMutation = useMutation({
    retry: false,
    mutationFn: async (url: string) => {
      await remove(url);
      return url;
    },
  });
  const form = useForm({
    defaultValues: { image: null as File | null },
    onSubmitMeta: { action: "upload" as "upload" | "remove" },
    validators: {
      onSubmit: ({ value: draft }): string | undefined => {
        if (draft.image && draft.image.size > MAX_FILE_SIZE) return "The image size cannot exceed 5 MB.";
        if (draft.image && !Object.hasOwn(ACCEPTED_AVATAR_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE, draft.image.type)) {
          return "Please upload a file in a valid format.";
        }
        return undefined;
      },
    },
    onSubmit: async ({ value: draft, meta }) => {
      if (meta.action === "remove") {
        if (!value) return;
        if (removeMutation.data !== value) await removeMutation.mutateAsync(value);
        await handleRemove();
      } else {
        if (!draft.image) return;
        const result =
          uploadMutation.data?.file === draft.image
            ? uploadMutation.data
            : await uploadMutation.mutateAsync(draft.image);
        await onSuccess(result.url);
        request.current?.signal.throwIfAborted();
      }
      onClose();
    },
  });
  const { image, busy, errors } = useStore(form.store, (state) => ({
    image: state.values.image,
    busy: state.isSubmitting,
    errors: state.errors,
  }));
  const { getRootProps, getInputProps, isDragActive, fileRejections } = useDropzone({
    onDrop: (files) => {
      if (files[0]) form.setFieldValue("image", files[0]);
    },
    accept: ACCEPTED_AVATAR_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE,
    maxSize: MAX_FILE_SIZE,
    multiple: false,
    disabled: busy,
  });
  const submit = (action: "upload" | "remove") => {
    if (form.state.isSubmitting) return;
    void form
      .handleSubmit({ action })
      .catch(() => form.setErrorMap({ onSubmit: "The image could not be saved. Please try again." }));
  };
  const close = () => {
    if (!form.state.isSubmitting) onClose();
  };
  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.CENTER} width={EModalWidth.XL}>
      <form
        className="space-y-5 px-5 py-8 sm:p-6"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          event.stopPropagation();
          submit("upload");
        }}
      >
        <h3 className="text-16 leading-6 font-medium text-primary">Upload image</h3>
        <div className="space-y-3">
          <div className="flex items-center justify-center gap-3">
            <div
              {...getRootProps()}
              className={`relative grid h-80 w-80 cursor-pointer place-items-center rounded-lg p-12 text-center focus:ring-2 focus:ring-accent-strong focus:ring-offset-2 focus:outline-none ${
                (image === null && isDragActive) || !value
                  ? "border-2 border-dashed border-subtle hover:bg-surface-2"
                  : ""
              }`}
            >
              {image || value ? (
                <>
                  <span className="absolute top-0 right-0 z-40 translate-x-1/2 -translate-y-1/2 rounded-sm bg-surface-2 px-2 py-0.5 text-11 font-medium text-secondary">
                    Edit
                  </span>
                  <ImagePreview
                    file={image}
                    src={getFileURL(value ?? "")}
                    alt="Selected avatar"
                    className="absolute top-0 left-0 h-full w-full rounded-md object-cover"
                  />
                </>
              ) : (
                <div>
                  <UserCirclePropertyIcon className="mx-auto h-16 w-16 text-secondary" />
                  <span className="mt-2 block text-13 font-medium text-secondary">
                    {isDragActive ? "Drop image here to upload" : "Drag & drop image here"}
                  </span>
                </div>
              )}
              <input {...getInputProps({ "aria-label": "Choose image" })} />
            </div>
          </div>
          {fileRejections.length > 0 && (
            <p role="alert" className="text-13 text-danger-primary">
              {fileRejections[0].errors[0].code === "file-too-large"
                ? "The image size cannot exceed 5 MB."
                : "Please upload a file in a valid format."}
            </p>
          )}
          {errors.map((error) => (
            <p key={error} role="alert" className="text-13 text-danger-primary">
              {error}
            </p>
          ))}
        </div>
        <p className="my-4 text-13 text-secondary">File formats supported- .jpeg, .jpg, .png, .webp</p>
        <div className="flex items-center justify-between">
          <Button
            type="button"
            variant="error-fill"
            size="lg"
            disabled={!value || busy}
            onClick={() => {
              submit("remove");
            }}
          >
            Remove
          </Button>
          <div className="flex items-center gap-2">
            <Button type="button" variant="secondary" size="lg" onClick={close} disabled={busy}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" size="lg" disabled={!image || busy} loading={busy}>
              Upload & Save
            </Button>
          </div>
        </div>
      </form>
    </ModalCore>
  );
}
