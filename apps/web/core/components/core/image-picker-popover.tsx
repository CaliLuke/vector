/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState, useEffect, useRef, useMemo } from "react";

import { useDropzone } from "react-dropzone";
import { useForm, useStore } from "@tanstack/react-form";
import { skipToken, useQuery, useMutation } from "@tanstack/react-query";
import { Popover } from "@headlessui/react";
// plane imports
import { ACCEPTED_COVER_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE, MAX_FILE_SIZE } from "@plane/constants";
import { Tabs } from "@plane/propel/tabs";
import { Button, getButtonStyling } from "@plane/propel/button";
import { ImagePreview } from "@/components/core/image-preview";
import { EFileAssetType } from "@plane/types";
import { Input, Loader } from "@plane/ui";
// helpers
import { STATIC_COVER_IMAGES, getCoverImageDisplayURL } from "@/helpers/cover-image.helper";
// hooks
import { instanceConfigOptions } from "@/queries/instance";
// services
import { FileService } from "@/services/file.service";

type TTabOption = {
  key: string;
  title: string;
  isEnabled: boolean;
};

type Props = {
  label: string | React.ReactNode;
  value: string | null;
  onChange: (data: string) => void | Promise<void>;
  disabled?: boolean;
  tabIndex?: number;
} & (
  | { isProfileCover: true; workspaceSlug?: never; projectId?: never }
  | { isProfileCover?: false; workspaceSlug: string; projectId?: string | null }
);

// services
const fileService = new FileService();

export function ImagePickerPopover(props: Props) {
  return (
    <Popover className="relative z-19" tabIndex={props.tabIndex}>
      {({ close }) => (
        <>
          <Popover.Button type="button" className={getButtonStyling("secondary", "sm")} disabled={props.disabled}>
            {props.label}
          </Popover.Button>
          <Popover.Panel className="absolute right-0 z-20 mt-2 rounded-md border border-subtle bg-surface-1 shadow-raised-200">
            <ImagePickerContent
              key={props.isProfileCover ? "profile" : JSON.stringify([props.workspaceSlug, props.projectId])}
              {...props}
              close={close}
            />
          </Popover.Panel>
        </>
      )}
    </Popover>
  );
}

function ImagePickerContent(props: Props & { close: () => void }) {
  const { value, onChange, close } = props;
  const [searchParams, setSearchParams] = useState("");
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  const selection = useMutation({ mutationFn: async (url: string) => await onChange(url), retry: false });
  const upload = useMutation({
    retry: false,
    mutationFn: async (file: File) => {
      const controller = new AbortController();
      request.current = controller;
      const result = props.isProfileCover
        ? await fileService.uploadUserAsset(
            { entity_identifier: "", entity_type: EFileAssetType.USER_COVER },
            file,
            controller.signal
          )
        : await fileService.uploadWorkspaceAsset(
            props.workspaceSlug,
            { entity_identifier: props.projectId ?? "", entity_type: EFileAssetType.PROJECT_COVER },
            file,
            undefined,
            controller.signal
          );
      controller.signal.throwIfAborted();
      return { file, url: result.asset_url };
    },
  });
  const form = useForm({
    defaultValues: { image: null as File | null, search: "" },
    validators: {
      onSubmit: ({ value: draft }): string | undefined => {
        if (!draft.image) return "Choose an image to upload.";
        if (draft.image.size > MAX_FILE_SIZE) return "The image size cannot exceed 5 MB.";
        if (!Object.hasOwn(ACCEPTED_COVER_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE, draft.image.type))
          return "Please upload a file in a valid format.";
        return undefined;
      },
    },
    onSubmit: async ({ value: draft }) => {
      if (!draft.image) return;
      const result = upload.data?.file === draft.image ? upload.data : await upload.mutateAsync(draft.image);
      await selection.mutateAsync(result.url);
      request.current?.signal.throwIfAborted();
      close();
    },
  });
  const { image, search, isImageUploading, errors } = useStore(form.store, (state) => ({
    image: state.values.image,
    search: state.values.search,
    isImageUploading: state.isSubmitting,
    errors: state.errors,
  }));
  const busy = isImageUploading || selection.isPending;
  // store hooks
  const { data: config } = useQuery(instanceConfigOptions);
  // derived values
  const hasUnsplashConfigured = config?.has_unsplash_configured || false;
  const tabOptions: TTabOption[] = useMemo(
    () => [
      {
        key: "unsplash",
        title: "Unsplash",
        isEnabled: hasUnsplashConfigured,
      },
      {
        key: "images",
        title: "Images",
        isEnabled: true,
      },
      {
        key: "upload",
        title: "Upload",
        isEnabled: true,
      },
    ],
    [hasUnsplashConfigured]
  );

  const enabledTabs = useMemo(() => tabOptions.filter((tab) => tab.isEnabled), [tabOptions]);

  const {
    data: unsplashImages,
    error: unsplashError,
    refetch: retryUnsplash,
  } = useQuery({
    queryKey: ["UNSPLASH_IMAGES", searchParams],
    queryFn: hasUnsplashConfigured ? ({ signal }) => fileService.getUnsplashImages(searchParams, signal) : skipToken,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });

  return (
    <div className="flex h-96 w-80 flex-col overflow-auto rounded border border-subtle bg-surface-1 shadow-raised-200 md:h-[36rem] md:w-[36rem]">
      {selection.isError && (
        <p role="alert" className="p-3 text-13 text-danger-primary">
          The image could not be saved. Please try again.
        </p>
      )}
      {errors.map((error) => (
        <p role="alert" key={error} className="p-3 text-13 text-danger-primary">
          {error}
        </p>
      ))}
      <Tabs defaultValue={enabledTabs[0]?.key || "images"} className="flex h-full flex-col p-3">
        <Tabs.List className="flex rounded bg-layer-3 p-1">
          {enabledTabs.map((tab) => (
            <Tabs.Trigger key={tab.key} value={tab.key} size="md">
              {tab.title}
            </Tabs.Trigger>
          ))}
          <Tabs.Indicator />
        </Tabs.List>
        <div className="vertical-scrollbar mt-3 scrollbar-sm flex-1 overflow-x-hidden overflow-y-auto p-3">
          <Tabs.Content value="unsplash" className="h-full w-full space-y-4">
            <>
              <div className="flex items-center gap-x-2">
                <Input
                  id="search"
                  name="search"
                  type="text"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      setSearchParams(search);
                    }
                  }}
                  value={search}
                  onChange={(e) => form.setFieldValue("search", e.target.value)}
                  placeholder="Search for images"
                  className="w-full text-13"
                />
                <Button type="button" variant="primary" size="xl" onClick={() => setSearchParams(search)}>
                  Search
                </Button>
              </div>
              {unsplashError ? (
                <p role="alert" className="text-13 text-danger-primary">
                  Could not load images.
                  <Button type="button" variant="link" onClick={() => void retryUnsplash()}>
                    Retry search
                  </Button>
                </p>
              ) : unsplashImages ? (
                unsplashImages.length > 0 ? (
                  <div className="grid grid-cols-4 gap-4">
                    {unsplashImages.map((unsplashImage) => (
                      <button
                        type="button"
                        key={unsplashImage.id}
                        className="relative col-span-2 aspect-video md:col-span-1"
                        disabled={busy}
                        onClick={() => selection.mutate(unsplashImage.urls.regular, { onSuccess: () => close() })}
                      >
                        <img
                          src={unsplashImage.urls.small}
                          alt={unsplashImage.alt_description}
                          className="absolute top-0 left-0 h-full w-full cursor-pointer rounded-sm object-cover"
                        />
                      </button>
                    ))}
                  </div>
                ) : (
                  <p className="pt-7 text-center text-11 text-secondary">No images found.</p>
                )
              ) : (
                <Loader className="grid grid-cols-4 gap-4">
                  <Loader.Item height="80px" width="100%" />
                  <Loader.Item height="80px" width="100%" />
                  <Loader.Item height="80px" width="100%" />
                  <Loader.Item height="80px" width="100%" />
                  <Loader.Item height="80px" width="100%" />
                  <Loader.Item height="80px" width="100%" />
                  <Loader.Item height="80px" width="100%" />
                  <Loader.Item height="80px" width="100%" />
                </Loader>
              )}
            </>
          </Tabs.Content>
          <Tabs.Content value="images" className="h-full w-full space-y-4">
            <div className="grid grid-cols-4 gap-4">
              {Object.values(STATIC_COVER_IMAGES).map((imageUrl, index) => (
                <button
                  type="button"
                  key={imageUrl}
                  className="relative col-span-2 aspect-video md:col-span-1"
                  disabled={busy}
                  onClick={() => selection.mutate(imageUrl, { onSuccess: () => close() })}
                >
                  <img
                    src={imageUrl}
                    alt={`Cover ${index + 1}`}
                    className="absolute top-0 left-0 h-full w-full cursor-pointer rounded-sm object-cover transition-opacity hover:opacity-80"
                  />
                </button>
              ))}
            </div>
          </Tabs.Content>
          <Tabs.Content value="upload" className="h-full w-full">
            <div className="flex h-full w-full flex-col gap-y-2">
              <CoverUploadField
                image={image}
                value={value}
                busy={busy}
                onChange={(file) => form.setFieldValue("image", file)}
              />
              <div className="flex h-12 items-start justify-end gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  disabled={busy}
                  onClick={() => {
                    close();
                  }}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  variant="primary"
                  className="w-full"
                  onClick={() => {
                    if (!form.state.isSubmitting) {
                      void form
                        .handleSubmit()
                        .catch(() => form.setErrorMap({ onSubmit: "The image could not be saved. Please try again." }));
                    }
                  }}
                  disabled={!image || busy}
                  loading={isImageUploading}
                >
                  {isImageUploading ? "Uploading" : "Upload & Save"}
                </Button>
              </div>
            </div>
          </Tabs.Content>
        </div>
      </Tabs>
    </div>
  );
}

function CoverUploadField({
  image,
  value,
  busy,
  onChange,
}: {
  image: File | null;
  value: string | null;
  busy: boolean;
  onChange: (file: File) => void;
}) {
  const { getRootProps, getInputProps, isDragActive, fileRejections } = useDropzone({
    onDrop: (files) => {
      if (files[0]) onChange(files[0]);
    },
    accept: ACCEPTED_COVER_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE,
    maxSize: MAX_FILE_SIZE,
    multiple: false,
    disabled: busy,
  });

  return (
    <>
      <div className="flex w-full flex-1 items-center gap-3">
        <div
          {...getRootProps()}
          className={`relative grid h-full w-full cursor-pointer place-items-center rounded-lg p-12 text-center focus:ring-2 focus:ring-accent-strong focus:ring-offset-2 focus:outline-none ${
            (image === null && isDragActive) || !value ? "border-2 border-dashed border-subtle hover:bg-surface-2" : ""
          }`}
        >
          <span className="absolute top-0 right-0 z-40 -translate-y-1/2 rounded-sm bg-surface-2 px-2 py-0.5 text-11 font-medium text-secondary">
            Edit
          </span>
          {image || value ? (
            <>
              <ImagePreview
                file={image}
                src={getCoverImageDisplayURL(value, "")}
                alt="Selected cover"
                className="h-full w-full rounded-lg object-cover"
              />
            </>
          ) : (
            <div>
              <span className="mt-2 block text-13 font-medium text-secondary">
                {isDragActive ? "Drop image here to upload" : "Drag & drop image here"}
              </span>
            </div>
          )}

          <input {...getInputProps()} />
        </div>
      </div>
      {fileRejections.length > 0 && (
        <p className="text-13 text-danger-primary">
          {fileRejections[0].errors[0].code === "file-too-large"
            ? "The image size cannot exceed 5 MB."
            : "Please upload a file in a valid format."}
        </p>
      )}

      <p className="text-13 text-secondary">File formats supported- .jpeg, .jpg, .png, .webp</p>
    </>
  );
}
