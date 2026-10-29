/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Style } from "@react-pdf/types";
import { BACKGROUND_COLORS, CODE_COLORS, EDITOR_BACKGROUND_COLORS, LINK_COLORS, resolveColorForPdf } from "./colors";
import type { MarkRendererRegistry, TipTapMark } from "./types";

function resolveCustomColor(value: unknown, type: "text" | "background"): string | null {
  if (typeof value !== "string") return null;
  return resolveColorForPdf(value, type) ?? (value.startsWith("rgb") ? value : null);
}

const markRenderers: MarkRendererRegistry = {
  bold: (_mark: TipTapMark, style: Style): Style => ({
    ...style,
    fontWeight: "bold",
  }),

  italic: (_mark: TipTapMark, style: Style): Style => ({
    ...style,
    fontStyle: "italic",
  }),

  underline: (_mark: TipTapMark, style: Style): Style => ({
    ...style,
    textDecoration: "underline",
  }),

  strike: (_mark: TipTapMark, style: Style): Style => ({
    ...style,
    textDecoration: "line-through",
  }),

  code: (_mark: TipTapMark, style: Style): Style => ({
    ...style,
    fontFamily: "Courier",
    fontSize: 10,
    backgroundColor: BACKGROUND_COLORS.layer1,
    color: CODE_COLORS.text,
  }),

  link: (_mark: TipTapMark, style: Style): Style => ({
    ...style,
    color: LINK_COLORS.primary,
    textDecoration: "underline",
  }),

  textStyle: (mark: TipTapMark, style: Style): Style => {
    const attrs = mark.attrs || {};
    const newStyle: Style = { ...style };

    if (attrs.color && typeof attrs.color === "string") {
      newStyle.color = attrs.color;
    }

    if (attrs.backgroundColor && typeof attrs.backgroundColor === "string") {
      newStyle.backgroundColor = attrs.backgroundColor;
    }

    return newStyle;
  },

  highlight: (mark: TipTapMark, style: Style): Style => {
    const attrs = mark.attrs || {};
    return {
      ...style,
      backgroundColor: (attrs.color as string) || EDITOR_BACKGROUND_COLORS.purple,
    };
  },

  subscript: (_mark: TipTapMark, style: Style): Style => ({
    ...style,
    fontSize: 8,
  }),

  superscript: (_mark: TipTapMark, style: Style): Style => ({
    ...style,
    fontSize: 8,
  }),

  /**
   * Custom color mark handler
   * Handles the customColor extension which stores colors as data-text-color and data-background-color attributes
   * The colors can be either:
   * 1. Color keys like "gray", "peach", "pink", etc. (from COLORS_LIST)
   * 2. Direct hex values for custom colors
   * 3. CSS variable references like "var(--editor-colors-gray-text)"
   */
  customColor: (mark: TipTapMark, style: Style): Style => {
    const attrs = mark.attrs || {};
    const newStyle: Style = { ...style };

    const textColor = resolveCustomColor(attrs.color, "text");
    const backgroundColor = resolveCustomColor(attrs.backgroundColor, "background");
    if (textColor) newStyle.color = textColor;
    if (backgroundColor) newStyle.backgroundColor = backgroundColor;

    return newStyle;
  },
};

export const applyMarks = (marks: TipTapMark[] | undefined, baseStyle: Style = {}): Style => {
  if (!marks || marks.length === 0) {
    return baseStyle;
  }

  return marks.reduce((style, mark) => {
    const renderer = Object.hasOwn(markRenderers, mark.type) ? markRenderers[mark.type] : undefined;
    if (renderer) {
      return renderer(mark, style);
    }
    return style;
  }, baseStyle);
};
