/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { AlignCenter, AlignLeft, AlignRight } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { stringAttribute } from "@/helpers/node-attributes";
// local imports
import { ECustomImageAttributeNames, ECustomImageStatus } from "./types";
import type { TCustomImageAlignment, Pixel, TCustomImageAttributes } from "./types";

export const DEFAULT_CUSTOM_IMAGE_ATTRIBUTES: TCustomImageAttributes = {
  [ECustomImageAttributeNames.SOURCE]: null,
  [ECustomImageAttributeNames.ID]: null,
  [ECustomImageAttributeNames.WIDTH]: "35%",
  [ECustomImageAttributeNames.HEIGHT]: "auto",
  [ECustomImageAttributeNames.ASPECT_RATIO]: null,
  [ECustomImageAttributeNames.ALIGNMENT]: "left",
  [ECustomImageAttributeNames.STATUS]: ECustomImageStatus.PENDING,
};

function isPixel(value: unknown): value is Pixel {
  return typeof value === "string" && /^\d+(?:\.\d+)?px$/.test(value);
}

function imageDimension<T extends "35%" | "auto">(value: unknown, fallback: T): Pixel | number | T {
  if (isPixel(value)) return value;
  if (typeof value === "number" && Number.isFinite(value) && value > 0) return value;
  return fallback;
}

export function getImageAttributes(attributes: Record<string, unknown>): TCustomImageAttributes {
  const aspectRatio = attributes.aspectRatio;
  return {
    id: stringAttribute(attributes, "id") ?? null,
    src: stringAttribute(attributes, "src") ?? null,
    width: imageDimension(attributes.width, "35%"),
    height: imageDimension(attributes.height, "auto"),
    aspectRatio:
      typeof aspectRatio === "number" && Number.isFinite(aspectRatio) && aspectRatio > 0 ? aspectRatio : null,
    alignment: IMAGE_ALIGNMENT_OPTIONS.find((option) => option.value === attributes.alignment)?.value ?? "left",
    status:
      Object.values(ECustomImageStatus).find((status) => status === attributes.status) ?? ECustomImageStatus.PENDING,
  };
}

export const getImageComponentImageFileMap = (editor: Editor) => editor.storage.imageComponent?.fileMap;

export const ensurePixelString = <TDefault>(
  value: Pixel | TDefault | number | undefined | null,
  defaultValue?: TDefault
) => {
  if (!value || value === defaultValue) {
    return defaultValue;
  }

  if (typeof value === "number") {
    return `${value}px` satisfies Pixel;
  }

  return value;
};

export const IMAGE_ALIGNMENT_OPTIONS: {
  label: string;
  value: TCustomImageAlignment;
  icon: LucideIcon;
}[] = [
  {
    label: "Left",
    value: "left",
    icon: AlignLeft,
  },
  {
    label: "Center",
    value: "center",
    icon: AlignCenter,
  },
  {
    label: "Right",
    value: "right",
    icon: AlignRight,
  },
];
export const getImageBlockId = (id: string) => `editor-image-block-${id}`;

export const isImageDuplicating = (status: ECustomImageStatus) => status === ECustomImageStatus.DUPLICATING;

export const hasImageDuplicationFailed = (status: ECustomImageStatus) =>
  status === ECustomImageStatus.DUPLICATION_FAILED;
