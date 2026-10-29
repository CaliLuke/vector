/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ImageOptions } from "@tiptap/extension-image";
import type { Node } from "@tiptap/core";
// types

export enum ECustomImageAttributeNames {
  ID = "id",
  WIDTH = "width",
  HEIGHT = "height",
  ASPECT_RATIO = "aspectRatio",
  SOURCE = "src",
  ALIGNMENT = "alignment",
  STATUS = "status",
}

export type Pixel = `${number}px`;

type PixelAttribute<TDefault> = Pixel | TDefault;

export type TCustomImageSize = {
  width: PixelAttribute<"35%">;
  height: PixelAttribute<"auto">;
  aspectRatio: number | null;
};

export type TCustomImageAlignment = "left" | "center" | "right";

export enum ECustomImageStatus {
  PENDING = "pending",
  UPLOADING = "uploading",
  UPLOADED = "uploaded",
  DUPLICATING = "duplicating",
  DUPLICATION_FAILED = "duplication-failed",
}

export type TCustomImageAttributes = {
  [ECustomImageAttributeNames.ID]: string | null;
  [ECustomImageAttributeNames.WIDTH]: PixelAttribute<"35%" | number> | null;
  [ECustomImageAttributeNames.HEIGHT]: PixelAttribute<"auto" | number> | null;
  [ECustomImageAttributeNames.ASPECT_RATIO]: number | null;
  [ECustomImageAttributeNames.SOURCE]: string | null;
  [ECustomImageAttributeNames.ALIGNMENT]: TCustomImageAlignment;
  [ECustomImageAttributeNames.STATUS]: ECustomImageStatus;
};

type UploadEntity = ({ event: "insert" } | { event: "drop"; file: File }) & { hasOpenedFileInputOnce?: boolean };

export type InsertImageComponentProps = {
  file?: File;
  pos?: number;
  event: "insert" | "drop";
};

export type CustomImageExtensionStorage = {
  fileMap: Map<string, UploadEntity>;
  deletedImageSet: Map<string, boolean>;
  maxFileSize: number;
};

export type CustomImageExtensionType = Node<ImageOptions, CustomImageExtensionStorage>;
