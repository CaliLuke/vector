/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { NodeSelection } from "@tiptap/pm/state";
import React, { useRef, useState, useCallback, useMemo } from "react";
// plane imports
import { cn } from "@plane/utils/common";
// local imports
import { ECustomImageAttributeNames } from "../types";
import type { TCustomImageAttributes, TCustomImageSize } from "../types";
import { getImageAttributes, getImageBlockId, isImageDuplicating } from "../utils";
import type { CustomImageNodeViewProps } from "./node-view";
import { ImageToolbarRoot } from "./toolbar";
import { ImageUploadStatus } from "./upload-status";
import { useImageSize } from "./use-image-size";

type CustomImageBlockProps = CustomImageNodeViewProps & {
  editorContainer: HTMLDivElement | null;
  imageFromFileSystem: string | undefined;
  setEditorContainer: (editorContainer: HTMLDivElement | null) => void;
  setFailedToLoadImage: (isError: boolean) => void;
  src: string | undefined;
  downloadSrc: string | undefined;
};

export function CustomImageBlock(props: CustomImageBlockProps) {
  // props
  const {
    editor,
    editorContainer,
    getPos,
    imageFromFileSystem,
    node,
    selected,
    setEditorContainer,
    setFailedToLoadImage,
    src: resolvedImageSrc,
    downloadSrc: resolvedDownloadSrc,
    updateAttributes,
  } = props;
  const attrs = useMemo(() => getImageAttributes(node.attrs), [node.attrs]);
  const { src: imgNodeSrc, alignment: nodeAlignment, status } = attrs;
  const containerRef = useRef<HTMLDivElement>(null);
  const imageRef = useRef<HTMLImageElement>(null);
  const [hasErroredOnFirstLoad, setHasErroredOnFirstLoad] = useState(false);
  const [hasTriedRestoringImageOnce, setHasTriedRestoringImageOnce] = useState(false);
  // extension options
  const isTouchDevice = !!(editor.storage.utility as { isTouchDevice?: boolean } | undefined)?.isTouchDevice;

  const updateAttributesSafely = useCallback(
    (attributes: Partial<TCustomImageAttributes>, errorMessage: string) => {
      try {
        updateAttributes(attributes);
      } catch (error) {
        console.error(`${errorMessage}:`, error);
      }
    },
    [updateAttributes]
  );

  const { size, isResizing, initialResizeComplete, handleImageLoad, handleResizeStart, handleWidthChange } =
    useImageSize({
      attrs,
      editorContainer,
      containerRef,
      imageRef,
      setEditorContainer,
      updateAttributes: updateAttributesSafely,
    });

  const selectImage = useCallback(
    (e: React.MouseEvent<HTMLButtonElement>) => {
      e.stopPropagation();
      if (isTouchDevice) {
        e.preventDefault();
        editor.commands.blur();
      }
      const pos = getPos();
      if (pos === undefined) return;
      const nodeSelection = NodeSelection.create(editor.state.doc, pos);
      editor.view.dispatch(editor.state.tr.setSelection(nodeSelection));
    },
    [editor, getPos, isTouchDevice]
  );

  const isDuplicating = isImageDuplicating(status);
  // show the image loader if the remote image's src or preview image from filesystem is not set yet (while loading the image post upload) (or)
  // if the initial resize (from 35% width and "auto" height attrs to the actual size in px) is not complete
  const showImageLoader = !resolvedImageSrc || !initialResizeComplete || hasErroredOnFirstLoad || isDuplicating; // show the image upload status only when the resolvedImageSrc is not ready
  const showUploadStatus = !resolvedImageSrc && !isDuplicating;
  // show the preview image from the file system if the remote image's src is not set
  const displayedImageSrc = resolvedImageSrc || imageFromFileSystem;

  const imageStyle = { width: size.width, ...(size.aspectRatio && { aspectRatio: size.aspectRatio }) };

  return (
    <div
      id={attrs[ECustomImageAttributeNames.ID] ? getImageBlockId(attrs[ECustomImageAttributeNames.ID]) : undefined}
      className={cn("w-fit max-w-full transition-all", {
        "ml-[50%] -translate-x-1/2": nodeAlignment === "center",
        "ml-[100%] -translate-x-full": nodeAlignment === "right",
      })}
    >
      <div ref={containerRef} className="group/image-component relative inline-block max-w-full" style={imageStyle}>
        {showImageLoader && (
          <div className="animate-pulse rounded-md bg-layer-1" style={{ width: size.width, height: size.height }} />
        )}
        <button
          type="button"
          aria-label="Select image"
          onClick={selectImage}
          className="block max-w-full rounded-md focus-visible:outline-2 focus-visible:outline-accent-strong"
        >
          <img
            ref={imageRef}
            src={displayedImageSrc}
            alt=""
            onLoad={handleImageLoad}
            onError={(_e) =>
              void (async () => {
                if (hasTriedRestoringImageOnce) {
                  setFailedToLoadImage(true);
                  return;
                }

                try {
                  setHasErroredOnFirstLoad(true);
                  // this is a type error from tiptap, don't remove await until it's fixed
                  if (!imgNodeSrc) {
                    throw new Error("No source image to restore from");
                  }
                  await editor.storage.utility.fileHandler.restore(imgNodeSrc);
                  if (!imageRef.current) {
                    throw new Error("Image reference not found");
                  }
                  if (!resolvedImageSrc) {
                    throw new Error("No resolved image source available");
                  }
                  if (isTouchDevice) {
                    const refreshedSrc = await editor.storage.utility.fileHandler.getAssetSrc(imgNodeSrc);
                    imageRef.current.src = refreshedSrc;
                  } else {
                    imageRef.current.src = resolvedImageSrc;
                  }
                } catch (error) {
                  // if the image failed to even restore, then show the error state
                  setFailedToLoadImage(true);
                  console.error("Error while loading image", error);
                } finally {
                  setHasErroredOnFirstLoad(false);
                  setHasTriedRestoringImageOnce(true);
                }
              })()
            }
            width={size.width}
            className={cn("image-component block rounded-md", {
              // hide the image while the background calculations of the image loader are in progress (to avoid flickering) and show the loader until then
              hidden: showImageLoader,
              "read-only-image": !editor.isEditable,
              "loading-image opacity-80 blur-sm": !resolvedImageSrc,
            })}
            style={imageStyle}
          />
        </button>
        {showUploadStatus && attrs[ECustomImageAttributeNames.ID] && (
          <ImageUploadStatus editor={editor} nodeId={attrs[ECustomImageAttributeNames.ID]} />
        )}
        {selected && displayedImageSrc === resolvedImageSrc && (
          <div className="pointer-events-none absolute inset-0 size-full bg-accent-primary/30" />
        )}
        <ImageTools
          editor={editor}
          size={size}
          isResizing={isResizing}
          ready={initialResizeComplete && !isDuplicating}
          nodeAlignment={nodeAlignment}
          src={resolvedImageSrc}
          downloadSrc={resolvedDownloadSrc}
          editorContainer={editorContainer}
          isTouchDevice={isTouchDevice}
          updateAttributes={updateAttributesSafely}
          handleResizeStart={handleResizeStart}
          handleWidthChange={handleWidthChange}
        />
      </div>
    </div>
  );
}

type ImageToolsProps = {
  editor: CustomImageBlockProps["editor"];
  size: TCustomImageSize;
  isResizing: boolean;
  ready: boolean;
  nodeAlignment: TCustomImageAttributes["alignment"];
  src: string | undefined;
  downloadSrc: string | undefined;
  editorContainer: HTMLDivElement | null;
  isTouchDevice: boolean;
  updateAttributes: (attrs: Partial<TCustomImageAttributes>, message: string) => void;
  handleResizeStart: (event: React.MouseEvent | React.TouchEvent) => void;
  handleWidthChange: (width: number) => void;
};

function ImageTools({
  editor,
  size,
  isResizing,
  ready,
  nodeAlignment,
  src,
  downloadSrc,
  editorContainer,
  isTouchDevice,
  updateAttributes,
  handleResizeStart,
  handleWidthChange,
}: ImageToolsProps) {
  if (!ready || !src) return null;
  return (
    <>
      {downloadSrc && (
        <ImageToolbarRoot
          alignment={nodeAlignment ?? "left"}
          editor={editor}
          aspectRatio={size.aspectRatio === null ? 1 : size.aspectRatio}
          downloadSrc={downloadSrc}
          handleAlignmentChange={(alignment) =>
            updateAttributes({ alignment }, "Failed to update attributes while changing alignment:")
          }
          height={size.height}
          isTouchDevice={isTouchDevice}
          width={size.width}
          src={src}
        />
      )}
      {editor.isEditable && (
        <>
          <div
            className={cn(
              "pointer-events-none absolute inset-0 rounded-md border-2 border-accent-strong transition-opacity duration-100 ease-in-out",
              {
                "opacity-100": isResizing,
                "opacity-0 group-hover/image-component:opacity-100": !isResizing,
              }
            )}
          />
          <input
            type="range"
            aria-label="Image width"
            aria-valuetext={`${Math.round(parseFloat(size.width))} pixels`}
            min={100}
            max={Math.max(editorContainer?.clientWidth ?? 1000, parseFloat(size.width), 100)}
            step={1}
            value={parseFloat(size.width)}
            onChange={(event) => handleWidthChange(event.currentTarget.valueAsNumber)}
            className={cn(
              "absolute bottom-0 size-4 translate-y-1/2 appearance-none rounded-full border-2 border-white bg-accent-primary transition-opacity duration-100 ease-in-out focus-visible:opacity-100 focus-visible:outline-2 focus-visible:outline-accent-strong",
              {
                "pointer-events-auto opacity-100": isResizing,
                "pointer-events-none opacity-0 group-hover/image-component:pointer-events-auto group-hover/image-component:opacity-100":
                  !isResizing,
                "left-0 -translate-x-1/2 cursor-nesw-resize": nodeAlignment === "right",
                "right-0 translate-x-1/2 cursor-nwse-resize": nodeAlignment !== "right",
              }
            )}
            onMouseDown={handleResizeStart}
            onTouchStart={handleResizeStart}
          />
        </>
      )}
    </>
  );
}
