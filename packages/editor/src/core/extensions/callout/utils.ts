/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TLogoProps } from "@plane/types";
import { isRecord } from "@plane/utils/common";
import { stringAttribute } from "@/helpers/node-attributes";
import { sanitizeHTML } from "@plane/utils/string";
// types
import type { TCalloutBlockAttributes, TCalloutBlockEmojiAttributes, TCalloutBlockIconAttributes } from "./types";
import { ECalloutAttributeNames } from "./types";

export const DEFAULT_CALLOUT_BLOCK_ATTRIBUTES: TCalloutBlockAttributes = {
  [ECalloutAttributeNames.ID]: null,
  [ECalloutAttributeNames.LOGO_IN_USE]: "emoji",
  [ECalloutAttributeNames.ICON_COLOR]: undefined,
  [ECalloutAttributeNames.ICON_NAME]: undefined,
  [ECalloutAttributeNames.EMOJI_UNICODE]: "128161",
  [ECalloutAttributeNames.EMOJI_URL]: "https://cdn.jsdelivr.net/npm/emoji-datasource-apple/img/apple/64/1f4a1.png",
  [ECalloutAttributeNames.BACKGROUND]: undefined,
  [ECalloutAttributeNames.BLOCK_TYPE]: "callout-component",
};

type TStoredLogoValue = Pick<TCalloutBlockAttributes, ECalloutAttributeNames.LOGO_IN_USE> &
  (TCalloutBlockEmojiAttributes | TCalloutBlockIconAttributes);

export function getCalloutAttributes(attributes: Record<string, unknown>): TCalloutBlockAttributes {
  return {
    id: stringAttribute(attributes, "id") ?? null,
    "data-logo-in-use": attributes["data-logo-in-use"] === "icon" ? "icon" : "emoji",
    "data-icon-color": stringAttribute(attributes, "data-icon-color"),
    "data-icon-name": stringAttribute(attributes, "data-icon-name"),
    "data-emoji-unicode": stringAttribute(attributes, "data-emoji-unicode"),
    "data-emoji-url": stringAttribute(attributes, "data-emoji-url"),
    "data-background": stringAttribute(attributes, "data-background"),
    "data-block-type": "callout-component",
  };
}

function readStoredLogo(value: unknown): TStoredLogoValue | undefined {
  if (!isRecord(value)) return undefined;
  if (value.in_use === "emoji" && isRecord(value.emoji)) {
    const unicode = stringAttribute(value.emoji, "value");
    if (unicode)
      return {
        "data-logo-in-use": "emoji",
        "data-emoji-unicode": unicode,
        "data-emoji-url": stringAttribute(value.emoji, "url") || DEFAULT_CALLOUT_BLOCK_ATTRIBUTES["data-emoji-url"],
      };
  }
  if (value.in_use === "icon" && isRecord(value.icon)) {
    const name = stringAttribute(value.icon, "name");
    if (name)
      return {
        "data-logo-in-use": "icon",
        "data-icon-name": name,
        "data-icon-color": stringAttribute(value.icon, "color") || undefined,
      };
  }
  return undefined;
}

export const getStoredLogo = (): TStoredLogoValue => {
  const fallback: TStoredLogoValue = {
    "data-logo-in-use": "emoji",
    "data-emoji-unicode": DEFAULT_CALLOUT_BLOCK_ATTRIBUTES["data-emoji-unicode"],
    "data-emoji-url": DEFAULT_CALLOUT_BLOCK_ATTRIBUTES["data-emoji-url"],
  };
  if (typeof window === "undefined") return fallback;
  const stored = sanitizeHTML(localStorage.getItem("editor-calloutComponent-logo") ?? "");
  if (!stored) return fallback;
  try {
    const value: unknown = JSON.parse(stored);
    return readStoredLogo(value) ?? fallback;
  } catch (error) {
    console.error("Error parsing stored callout logo", error);
    localStorage.removeItem("editor-calloutComponent-logo");
    return fallback;
  }
};
// function to update the stored logo on local storage
export const updateStoredLogo = (value: TLogoProps): void => {
  if (typeof window === "undefined") return;
  localStorage.setItem("editor-calloutComponent-logo", JSON.stringify(value));
};
// function to get the stored background color from local storage
export const getStoredBackgroundColor = (): string | null => {
  if (typeof window !== "undefined") {
    return sanitizeHTML(localStorage.getItem("editor-calloutComponent-background") ?? "");
  }
  return null;
};
// function to update the stored background color on local storage
export const updateStoredBackgroundColor = (value: string | null): void => {
  if (typeof window === "undefined") return;
  if (value === null) {
    localStorage.removeItem("editor-calloutComponent-background");
    return;
  } else {
    localStorage.setItem("editor-calloutComponent-background", value);
  }
};
