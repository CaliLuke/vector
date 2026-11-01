/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Download, Minus } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ModalPortal } from "@plane/propel/portal";
import { NewTabIcon, PlusIcon, CloseIcon } from "@plane/propel/icons";
// plane imports
import { cn } from "@plane/utils/common";

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 2;
const ZOOM_SPEED = 0.05;
const ZOOM_STEPS = [0.5, 1, 1.5, 2];
const PAN_OFFSETS = new Map<string, [number, number]>([
  ["ArrowLeft", [-40, 0]],
  ["ArrowRight", [40, 0]],
  ["ArrowUp", [0, -40]],
  ["ArrowDown", [0, 40]],
]);

type Props = {
  aspectRatio: number;
  downloadSrc: string;
  isFullScreenEnabled: boolean;
  isTouchDevice: boolean;
  src: string;
  toggleFullScreenMode: (val: boolean) => void;
  width: string;
};

export function ImageFullScreenModal(props: Props) {
  const { aspectRatio, isFullScreenEnabled, isTouchDevice, downloadSrc, src, toggleFullScreenMode, width } = props;
  // refs
  const dragStart = useRef({ x: 0, y: 0 });
  const dragOffset = useRef({ x: 0, y: 0 });

  const [magnification, setMagnification] = useState<number>(1);
  const [initialMagnification, setInitialMagnification] = useState(1);
  const [isDragging, setIsDragging] = useState(false);
  const imgRef = useRef<HTMLImageElement | null>(null);

  const widthInNumber = useMemo(() => {
    if (!width) return 0;
    return Number(width.replace("px", ""));
  }, [width]);

  const setImageRef = useCallback(
    (node: HTMLImageElement | null) => {
      if (!node || !isFullScreenEnabled) return;

      imgRef.current = node;

      const viewportWidth = window.innerWidth * 0.9;
      const viewportHeight = window.innerHeight * 0.75;
      const imageWidth = widthInNumber;
      const imageHeight = imageWidth / aspectRatio;

      const widthRatio = viewportWidth / imageWidth;
      const heightRatio = viewportHeight / imageHeight;

      setInitialMagnification(Math.min(widthRatio, heightRatio));
      setMagnification(1);

      // Reset image position
      node.style.left = "0px";
      node.style.top = "0px";
    },
    [isFullScreenEnabled, widthInNumber, aspectRatio]
  );

  const handleClose = useCallback(() => {
    if (isDragging) return;
    toggleFullScreenMode(false);
    setMagnification(1);
    setInitialMagnification(1);
  }, [isDragging, toggleFullScreenMode]);

  const handleMagnification = useCallback((direction: "increase" | "decrease") => {
    setMagnification((prev) => {
      // Find the appropriate target zoom level based on current magnification
      let targetZoom: number;
      if (direction === "increase") {
        targetZoom = ZOOM_STEPS.find((step) => step > prev) ?? MAX_ZOOM;
      } else {
        // Reverse the array to find the next lower step
        targetZoom = ZOOM_STEPS.toReversed().find((step) => step < prev) ?? MIN_ZOOM;
      }

      // Reset position when zoom matches initial magnification
      if (targetZoom === 1 && imgRef.current) {
        imgRef.current.style.left = "0px";
        imgRef.current.style.top = "0px";
      }

      return targetZoom;
    });
  }, []);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "+" || e.key === "=" || e.key === "-") {
        e.preventDefault();
        e.stopPropagation();

        if (e.key === "+" || e.key === "=") handleMagnification("increase");
        if (e.key === "-") handleMagnification("decrease");
      }
    },
    [handleMagnification]
  );

  const handleMouseDown = (e: React.MouseEvent) => {
    if (!imgRef.current) return;

    const imgWidth = imgRef.current.offsetWidth * magnification;
    const imgHeight = imgRef.current.offsetHeight * magnification;
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    if (imgWidth > viewportWidth || imgHeight > viewportHeight) {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(true);
      dragStart.current = { x: e.clientX, y: e.clientY };
      dragOffset.current = {
        x: parseInt(imgRef.current.style.left || "0"),
        y: parseInt(imgRef.current.style.top || "0"),
      };
    }
  };

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging || !imgRef.current) return;

      const dx = e.clientX - dragStart.current.x;
      const dy = e.clientY - dragStart.current.y;

      // Apply the scale factor to the drag movement
      const scaledDx = dx / magnification;
      const scaledDy = dy / magnification;

      imgRef.current.style.left = `${dragOffset.current.x + scaledDx}px`;
      imgRef.current.style.top = `${dragOffset.current.y + scaledDy}px`;
    },
    [isDragging, magnification]
  );

  const handleMouseUp = useCallback(() => {
    if (!isDragging || !imgRef.current) return;
    setIsDragging(false);
  }, [isDragging]);

  const handleWheel = useCallback(
    (e: WheelEvent) => {
      if (!imgRef.current || !isFullScreenEnabled) return;

      e.preventDefault();

      // Handle pinch-to-zoom
      if (e.ctrlKey || e.metaKey) {
        const delta = e.deltaY;
        setMagnification((prev) => {
          const newZoom = prev * (1 - delta * ZOOM_SPEED);
          const clampedZoom = Math.min(Math.max(newZoom, MIN_ZOOM), MAX_ZOOM);

          // Reset position when zoom matches initial magnification
          if (clampedZoom === 1 && imgRef.current) {
            imgRef.current.style.left = "0px";
            imgRef.current.style.top = "0px";
          }

          return clampedZoom;
        });
        return;
      }
    },
    [isFullScreenEnabled]
  );

  // Event listeners
  useEffect(() => {
    if (!isFullScreenEnabled) return undefined;

    document.addEventListener("keydown", handleKeyDown);
    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
    window.addEventListener("wheel", handleWheel, { passive: false });

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
      window.removeEventListener("wheel", handleWheel);
    };
  }, [isFullScreenEnabled, handleKeyDown, handleMouseMove, handleMouseUp, handleWheel]);

  if (!isFullScreenEnabled) return null;

  return (
    <ModalPortal
      isOpen={isFullScreenEnabled}
      onClose={handleClose}
      portalId="editor-portal"
      fullScreen
      zIndex={50}
      ariaLabel="Fullscreen image viewer"
      overlayClassName="bg-black/90"
      contentClassName="pointer-events-none grid place-items-center overflow-hidden bg-transparent shadow-none"
    >
      <div
        className={cn("relative grid size-full place-items-center overflow-hidden", { "cursor-grabbing": isDragging })}
      >
        <button
          type="button"
          onClick={handleClose}
          className="pointer-events-auto absolute top-10 right-10 z-10 grid size-8 place-items-center"
          aria-label="Close image viewer"
        >
          <CloseIcon className="size-8 text-white/60 transition-colors hover:text-white" />
        </button>
        <button
          type="button"
          aria-label="Pan image with arrow keys; press Enter to center"
          className="pointer-events-auto rounded-lg focus-visible:outline-2 focus-visible:outline-white"
          onMouseDown={handleMouseDown}
          onClick={(event) => {
            if (event.detail === 0 && imgRef.current) {
              imgRef.current.style.left = "0px";
              imgRef.current.style.top = "0px";
            }
          }}
          onKeyDown={(event) => {
            const direction = PAN_OFFSETS.get(event.key);
            const image = imgRef.current;
            if (!direction || !image) return;
            event.preventDefault();
            image.style.left = `${parseFloat(image.style.left || "0") + direction[0] / magnification}px`;
            image.style.top = `${parseFloat(image.style.top || "0") + direction[1] / magnification}px`;
          }}
        >
          <img
            alt="Expanded attachment"
            draggable={false}
            ref={setImageRef}
            src={src}
            className="read-only-image rounded-lg"
            style={{
              width: `${widthInNumber * initialMagnification}px`,
              maxWidth: "none",
              maxHeight: "none",
              aspectRatio,
              position: "relative",
              transform: `scale(${magnification})`,
              transformOrigin: "center",
              transition: "width 0.2s ease, transform 0.2s ease",
            }}
          />
        </button>
        <div className="pointer-events-auto fixed bottom-10 left-1/2 flex -translate-x-1/2 items-center justify-center gap-1 divide-x divide-subtle-1 rounded-md border border-subtle-1 bg-black py-2">
          <div className="flex items-center">
            <button
              type="button"
              onClick={(e) => {
                if (isTouchDevice) {
                  e.preventDefault();
                  e.stopPropagation();
                }
                handleMagnification("decrease");
              }}
              className="grid size-6 place-items-center text-white/60 transition-colors duration-200 hover:text-white disabled:text-white/30"
              disabled={magnification <= MIN_ZOOM}
              aria-label="Zoom out"
            >
              <Minus className="size-4" />
            </button>
            <span className="w-12 text-center text-13 text-white">{Math.round(100 * magnification)}%</span>
            <button
              type="button"
              onClick={(e) => {
                if (isTouchDevice) {
                  e.preventDefault();
                  e.stopPropagation();
                }
                handleMagnification("increase");
              }}
              className="grid size-6 place-items-center text-white/60 transition-colors duration-200 hover:text-white disabled:text-white/30"
              disabled={magnification >= MAX_ZOOM}
              aria-label="Zoom in"
            >
              <PlusIcon className="size-4" />
            </button>
          </div>
          {!isTouchDevice && (
            <button
              type="button"
              onClick={() => window.open(downloadSrc, "_blank")}
              className="grid size-8 flex-shrink-0 place-items-center text-white/60 transition-colors duration-200 hover:text-white"
              aria-label="Download image"
            >
              <Download className="size-4" />
            </button>
          )}
          {!isTouchDevice && (
            <button
              type="button"
              onClick={() => window.open(src, "_blank")}
              className="grid size-8 flex-shrink-0 place-items-center text-white/60 transition-colors duration-200 hover:text-white"
              aria-label="Open image in new tab"
            >
              <NewTabIcon className="size-4" />
            </button>
          )}
        </div>
      </div>
    </ModalPortal>
  );
}
