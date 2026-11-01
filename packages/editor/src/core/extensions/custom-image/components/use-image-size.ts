/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import type { MouseEvent as ReactMouseEvent, RefObject, TouchEvent as ReactTouchEvent } from "react";
import type { TCustomImageAttributes, TCustomImageSize } from "../types";
import { ensurePixelString } from "../utils";

const MIN_WIDTH = 100;
type Props = {
  attrs: TCustomImageAttributes;
  editorContainer: HTMLDivElement | null;
  containerRef: RefObject<HTMLDivElement | null>;
  imageRef: RefObject<HTMLImageElement | null>;
  setEditorContainer: (container: HTMLDivElement) => void;
  updateAttributes: (attributes: Partial<TCustomImageAttributes>, errorMessage: string) => void;
};
type Resize = { source: TCustomImageAttributes; size: TCustomImageSize; bounds: DOMRect };

function savedSize(attrs: TCustomImageAttributes): TCustomImageSize {
  return {
    width: ensurePixelString(attrs.width, "35%") ?? "35%",
    height: ensurePixelString(attrs.height, "auto") ?? "auto",
    aspectRatio: attrs.aspectRatio,
  };
}

function sizeAtWidth(width: number, aspectRatio: number): TCustomImageSize {
  const clamped = Math.max(width, MIN_WIDTH);
  return { width: `${clamped}px`, height: `${clamped / aspectRatio}px`, aspectRatio };
}

// Saved node attributes own the size; only an active gesture has a local draft.
export function useImageSize({
  attrs,
  editorContainer,
  containerRef,
  imageRef,
  setEditorContainer,
  updateAttributes,
}: Props) {
  const [draft, setDraft] = useState<Resize | null>(null);
  const gesture = useRef<Resize | null>(null);
  const [initialResizeComplete, setInitialResizeComplete] = useState(false);
  const size = draft?.source === attrs ? draft.size : savedSize(attrs);
  const isResizing = draft?.source === attrs;

  const handleImageLoad = useCallback(() => {
    const img = imageRef.current;
    if (!img) return;
    const container = editorContainer ?? img.closest<HTMLDivElement>(".editor-container");
    if (!container) {
      console.error("Editor container not found");
      return;
    }
    const aspectRatio = img.naturalWidth / img.naturalHeight;
    if (!Number.isFinite(aspectRatio) || aspectRatio <= 0) return;
    setEditorContainer(container);
    if (attrs.width === "35%") {
      const width = Math.round(Math.max(container.clientWidth * 0.35, MIN_WIDTH));
      updateAttributes(
        { width: `${width}px`, height: `${Math.round(width / aspectRatio)}px`, aspectRatio },
        "Failed to initialize image dimensions"
      );
    } else if (attrs.aspectRatio !== aspectRatio) {
      updateAttributes({ aspectRatio }, "Failed to initialize image aspect ratio");
    }
    setInitialResizeComplete(true);
  }, [attrs, editorContainer, imageRef, setEditorContainer, updateAttributes]);

  const handleResizeStart = useCallback(
    (event: ReactMouseEvent | ReactTouchEvent) => {
      event.preventDefault();
      event.stopPropagation();
      if (!containerRef.current || !attrs.aspectRatio) return;
      const next = { source: attrs, size: savedSize(attrs), bounds: containerRef.current.getBoundingClientRect() };
      gesture.current = next;
      setDraft(next);
    },
    [attrs, containerRef]
  );

  useEffect(() => {
    if (!isResizing) return undefined;
    const move = (event: MouseEvent | TouchEvent) => {
      const current = gesture.current;
      if (!current || current.source !== attrs || !attrs.aspectRatio) return;
      const point = "touches" in event ? event.touches[0] : event;
      if (!point) return;
      const width =
        attrs.alignment === "right" ? current.bounds.right - point.clientX : point.clientX - current.bounds.left;
      const next = { ...current, size: sizeAtWidth(width, attrs.aspectRatio) };
      gesture.current = next;
      setDraft(next);
    };
    const end = () => {
      const current = gesture.current;
      gesture.current = null;
      setDraft(null);
      if (current?.source === attrs) updateAttributes(current.size, "Failed to save resized image dimensions");
    };
    const cancel = () => {
      gesture.current = null;
      setDraft(null);
    };
    window.addEventListener("mousemove", move);
    window.addEventListener("touchmove", move);
    window.addEventListener("mouseup", end);
    window.addEventListener("mouseleave", end);
    window.addEventListener("touchend", end);
    window.addEventListener("touchcancel", cancel);
    return () => {
      window.removeEventListener("mousemove", move);
      window.removeEventListener("touchmove", move);
      window.removeEventListener("mouseup", end);
      window.removeEventListener("mouseleave", end);
      window.removeEventListener("touchend", end);
      window.removeEventListener("touchcancel", cancel);
    };
  }, [attrs, isResizing, updateAttributes]);

  const handleWidthChange = (width: number) => {
    if (!attrs.aspectRatio || !Number.isFinite(width)) return;
    updateAttributes(sizeAtWidth(width, attrs.aspectRatio), "Failed to save image width");
  };
  return { size, isResizing, initialResizeComplete, handleImageLoad, handleResizeStart, handleWidthChange };
}
