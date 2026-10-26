/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { NodeViewWrapper } from "@tiptap/react";
import type { NodeViewProps } from "@tiptap/react";
import { useEffect, useRef, useState } from "react";
// local imports
import type { CustomImageExtensionType, TCustomImageAttributes } from "../types";
import { ECustomImageAttributeNames, ECustomImageStatus } from "../types";
import { hasImageDuplicationFailed } from "../utils";
import { CustomImageBlock } from "./block";
import { CustomImageUploader } from "./uploader";

export type CustomImageNodeViewProps = Omit<NodeViewProps, "extension" | "updateAttributes"> & {
  extension: CustomImageExtensionType;
  node: NodeViewProps["node"] & {
    attrs: TCustomImageAttributes;
  };
  updateAttributes: (attrs: Partial<TCustomImageAttributes>) => void;
};

export function CustomImageNodeView(props: CustomImageNodeViewProps) {
  const { editor, node, updateAttributes, getPos } = props;
  const { src: imgNodeSrc, status } = node.attrs;

  const [isUploaded, setIsUploaded] = useState(!!imgNodeSrc);
  const [resolvedSrc, setResolvedSrc] = useState<string | undefined>(undefined);
  const [resolvedDownloadSrc, setResolvedDownloadSrc] = useState<string | undefined>(undefined);
  const [imageFromFileSystem, setImageFromFileSystem] = useState<string | undefined>(undefined);
  const [failedToLoadImage, setFailedToLoadImage] = useState(false);

  const [editorContainer, setEditorContainer] = useState<HTMLDivElement | null>(null);
  const imageComponentRef = useRef<HTMLDivElement>(null);
  const hasRetriedOnMount = useRef(false);

  useEffect(() => {
    const closestEditorContainer = imageComponentRef.current?.closest(".editor-container");
    if (closestEditorContainer) {
      setEditorContainer(closestEditorContainer as HTMLDivElement);
    }
  }, []);

  // the image is already uploaded if the image-component node has src attribute
  // and we need to remove the blob from our file system
  useEffect(() => {
    if (resolvedSrc || imgNodeSrc) {
      setIsUploaded(true);
      setImageFromFileSystem(undefined);
    } else {
      setIsUploaded(false);
    }
  }, [resolvedSrc, imgNodeSrc]);

  useEffect(() => {
    if (!imgNodeSrc) {
      setResolvedSrc(undefined);
      setResolvedDownloadSrc(undefined);
      return undefined;
    }

    setResolvedSrc(undefined);
    setResolvedDownloadSrc(undefined);
    setFailedToLoadImage(false);

    let current = true;
    const getImageSource = async () => {
      try {
        const url = await editor.storage.utility.fileHandler.getAssetSrc(imgNodeSrc);
        if (!current) return;
        setResolvedSrc(url);
        const downloadUrl = await editor.storage.utility.fileHandler.getAssetDownloadSrc(imgNodeSrc);
        if (!current) return;
        setResolvedDownloadSrc(downloadUrl);
      } catch (error) {
        if (!current) return;
        console.error("Error fetching image source:", error);
        setFailedToLoadImage(true);
      }
    };
    void getImageSource();
    return () => {
      current = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editor, imgNodeSrc]);

  useEffect(() => {
    let current = true;
    const ownsPendingCopy = () => {
      if (!current || editor.isDestroyed) return false;
      const position = getPos();
      const live = position === undefined ? undefined : editor.state.doc.nodeAt(position);
      return (
        live?.attrs.id === node.attrs.id &&
        live?.attrs.src === imgNodeSrc &&
        live?.attrs.status === ECustomImageStatus.DUPLICATING
      );
    };
    const handleDuplication = async () => {
      if (status !== ECustomImageStatus.DUPLICATING || !imgNodeSrc) {
        return;
      }

      try {
        hasRetriedOnMount.current = true;

        const newAssetId = await editor.storage.utility.fileHandler.duplicate(imgNodeSrc);

        if (!newAssetId) {
          throw new Error("Duplication returned invalid asset ID");
        }
        if (!ownsPendingCopy()) return;
        setFailedToLoadImage(false);
        updateAttributes({ src: newAssetId, status: ECustomImageStatus.UPLOADED });
      } catch (error: unknown) {
        if (!ownsPendingCopy()) return;
        console.error("Failed to duplicate image:", error);
        // Update status to failed
        updateAttributes({ status: ECustomImageStatus.DUPLICATION_FAILED });
      }
    };

    void handleDuplication();
    return () => {
      current = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editor, status, imgNodeSrc, node.attrs.id, getPos, updateAttributes]);

  useEffect(() => {
    if (hasImageDuplicationFailed(status) && !hasRetriedOnMount.current && imgNodeSrc) {
      hasRetriedOnMount.current = true;
      // Add a small delay before retrying to avoid immediate retries
      updateAttributes({ status: ECustomImageStatus.DUPLICATING });
    }
  }, [status, imgNodeSrc, updateAttributes]);

  useEffect(() => {
    if (status === ECustomImageStatus.UPLOADED) {
      hasRetriedOnMount.current = false;
      setFailedToLoadImage(false);
    }
  }, [status]);

  const hasDuplicationFailed = hasImageDuplicationFailed(status);
  const hasValidImageSource = imageFromFileSystem || (isUploaded && resolvedSrc);
  const shouldShowBlock = hasValidImageSource && !failedToLoadImage && !hasDuplicationFailed;

  return (
    <NodeViewWrapper key={node.attrs[ECustomImageAttributeNames.ID]}>
      <div className="mx-0 my-2 p-0" data-drag-handle ref={imageComponentRef}>
        {shouldShowBlock && !hasDuplicationFailed ? (
          <CustomImageBlock
            editorContainer={editorContainer}
            imageFromFileSystem={imageFromFileSystem}
            setEditorContainer={setEditorContainer}
            setFailedToLoadImage={setFailedToLoadImage}
            src={resolvedSrc}
            downloadSrc={resolvedDownloadSrc}
            {...props}
          />
        ) : (
          <CustomImageUploader
            failedToLoadImage={failedToLoadImage}
            hasDuplicationFailed={hasDuplicationFailed}
            loadImageFromFileSystem={setImageFromFileSystem}
            maxFileSize={(editor.storage.imageComponent as { maxFileSize?: number } | undefined)?.maxFileSize ?? 0}
            setIsUploaded={setIsUploaded}
            {...props}
          />
        )}
      </div>
    </NodeViewWrapper>
  );
}
