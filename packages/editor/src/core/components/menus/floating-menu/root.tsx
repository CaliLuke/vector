/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { FloatingFocusManager, FloatingOverlay, FloatingPortal } from "@floating-ui/react";
import type { UseInteractionsReturn, UseFloatingReturn } from "@floating-ui/react";

type Props = {
  children: React.ReactNode;
  classNames?: {
    buttonContainer?: string;
    button?: string;
  };
  getFloatingProps: UseInteractionsReturn["getFloatingProps"];
  getReferenceProps: UseInteractionsReturn["getReferenceProps"];
  menuButton: React.ReactNode;
  onClick?: (e: React.MouseEvent<HTMLButtonElement>) => void;
  options: UseFloatingReturn;
};

export function FloatingMenuRoot(props: Props) {
  const { children, classNames, getFloatingProps, getReferenceProps, menuButton, onClick, options } = props;
  // derived values
  const { refs, floatingStyles, context } = options;

  return (
    <>
      <div className={classNames?.buttonContainer}>
        <button
          ref={(node) => refs.setReference(node)}
          {...getReferenceProps()}
          type="button"
          className={classNames?.button}
          onClick={(e) => {
            context.onOpenChange(!context.open);
            onClick?.(e);
          }}
        >
          {menuButton}
        </button>
      </div>
      {context.open && (
        <FloatingPortal>
          {/* Backdrop */}
          <FloatingOverlay
            style={{
              zIndex: 99,
            }}
            lockScroll
          />
          <FloatingFocusManager context={context} restoreFocus>
            <div
              ref={(node) => refs.setFloating(node)}
              {...getFloatingProps()}
              style={{
                ...floatingStyles,
                zIndex: 100,
              }}
            >
              {children}
            </div>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}
