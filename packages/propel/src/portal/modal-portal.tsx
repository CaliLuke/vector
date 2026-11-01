/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import { cn } from "@plane/utils/common";
import {
  EPortalWidth,
  EPortalPosition,
  PORTAL_WIDTH_CLASSES,
  PORTAL_POSITION_CLASSES,
  DEFAULT_PORTAL_ID,
  MODAL_Z_INDEX,
} from "./constants";
import { PortalWrapper } from "./portal-wrapper";
import type { ModalPortalProps } from "./types";

/**
 * @param children - The modal content to render
 * @param isOpen - Whether the modal is open
 * @param onClose - Function to call when modal should close
 * @param portalId - The ID of the DOM element to render into
 * @param className - Custom className for the modal container
 * @param overlayClassName - Custom className for the overlay
 * @param contentClassName - Custom className for the content area
 * @param width - Predefined width options using EPortalWidth enum
 * @param position - Position of the modal using EPortalPosition enum
 * @param fullScreen - Whether to render in fullscreen mode
 * @param showOverlay - Whether to show background overlay
 * @param closeOnOverlayClick - Whether clicking overlay closes modal
 * @param closeOnEscape - Whether pressing Escape closes modal
 */
export function ModalPortal({
  children,
  isOpen,
  onClose,
  portalId = DEFAULT_PORTAL_ID,
  className,
  overlayClassName,
  contentClassName,
  width = EPortalWidth.HALF,
  position = EPortalPosition.RIGHT,
  fullScreen = false,
  showOverlay = true,
  closeOnOverlayClick = true,
  closeOnEscape = true,
  ariaLabel = "Dialog",
  zIndex = MODAL_Z_INDEX,
}: ModalPortalProps) {
  const modalClasses = cn(
    "shadow-lg absolute top-0 h-full bg-white transition-transform duration-300 ease-out",
    fullScreen ? "h-full w-full" : PORTAL_WIDTH_CLASSES[width],
    fullScreen ? "" : PORTAL_POSITION_CLASSES[position],
    contentClassName
  );
  if (!isOpen) return null;

  return (
    <BaseDialog.Root
      open={isOpen}
      disablePointerDismissal={!closeOnOverlayClick || !showOverlay}
      onOpenChange={(open, details) => {
        if (open) return;
        if (!onClose || (!closeOnEscape && details.reason === "escape-key")) {
          details.cancel();
          return;
        }
        onClose();
      }}
    >
      <PortalWrapper portalId={portalId}>
        <ModalLayer portalId={portalId} className={className} zIndex={zIndex}>
          {showOverlay && (
            <BaseDialog.Backdrop
              className={cn("absolute inset-0 bg-black/50 transition-colors duration-300", overlayClassName)}
            />
          )}
          <BaseDialog.Popup className={modalClasses} style={{ zIndex: zIndex + 1 }} aria-label={ariaLabel}>
            {children}
          </BaseDialog.Popup>
        </ModalLayer>
      </PortalWrapper>
    </BaseDialog.Root>
  );
}

// PortalWrapper owns the shared DOM container and renders this layer only after
// it exists. Base UI owns modal focus, nested dismissal, and scroll locking.
function ModalLayer({
  zIndex,
  portalId,
  className,
  children,
}: {
  zIndex: number;
  portalId: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <BaseDialog.Portal
      container={document.getElementById(portalId)}
      className={cn("fixed inset-0 h-full w-full overflow-y-auto", className)}
      style={{ zIndex }}
    >
      {children}
    </BaseDialog.Portal>
  );
}
