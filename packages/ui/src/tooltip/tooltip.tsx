/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Tooltip as BlueprintTooltip } from "@blueprintjs/core";
import { cloneElement } from "react";
import type {
  FocusEventHandler,
  MouseEventHandler,
  AriaAttributes,
  ReactElement,
  ReactNode,
  Ref,
  RefCallback,
  SyntheticEvent,
} from "react";
// helpers
import { cn } from "../utils/classname";

export type TPosition =
  | "top"
  | "right"
  | "bottom"
  | "left"
  | "auto"
  | "auto-end"
  | "auto-start"
  | "bottom-left"
  | "bottom-right"
  | "left-bottom"
  | "left-top"
  | "right-bottom"
  | "right-top"
  | "top-left"
  | "top-right";

interface ITooltipProps {
  tooltipHeading?: string;
  tooltipContent: string | ReactNode;
  position?: TPosition;
  children: ReactElement;
  disabled?: boolean;
  className?: string;
  openDelay?: number;
  closeDelay?: number;
  isMobile?: boolean;
  /** @deprecated The overlay is lazy; target handlers always mount for keyboard access. */
  renderByDefault?: boolean;
}

type TooltipTargetProps = {
  "aria-expanded"?: boolean;
  "aria-haspopup"?: AriaAttributes["aria-haspopup"];
  className?: string;
  onBlur?: FocusEventHandler<HTMLElement>;
  onContextMenu?: MouseEventHandler<HTMLElement>;
  onFocus?: FocusEventHandler<HTMLElement>;
  onMouseEnter?: MouseEventHandler<HTMLElement>;
  onMouseLeave?: MouseEventHandler<HTMLElement>;
  ref?: Ref<HTMLElement>;
  tabIndex?: number;
};

function mergeRefs<T>(...refs: Array<Ref<T> | undefined>): RefCallback<T> {
  return (node) => {
    const cleanups: Array<() => void> = [];
    const callbackRefs: Array<RefCallback<T>> = [];

    refs.forEach((ref) => {
      if (typeof ref === "function") {
        const cleanup = ref(node);
        if (typeof cleanup === "function") cleanups.push(cleanup);
        else callbackRefs.push(ref);
      } else if (ref) {
        ref.current = node;
        cleanups.push(() => {
          ref.current = null;
        });
      }
    });

    if (node === null) return;

    return () => {
      cleanups.forEach((cleanup) => cleanup());
      callbackRefs.forEach((ref) => ref(null));
    };
  };
}

function compose<T extends SyntheticEvent<HTMLElement>>(
  childHandler: ((event: T) => void) | undefined,
  tooltipHandler: ((event: T) => void) | undefined
): ((event: T) => void) | undefined {
  if (!childHandler) return tooltipHandler;
  if (!tooltipHandler) return childHandler;
  return (event) => {
    childHandler(event);
    tooltipHandler(event);
  };
}

export function Tooltip({
  tooltipHeading,
  tooltipContent,
  position = "top",
  children,
  disabled = false,
  className = "",
  openDelay = 200,
  closeDelay,
  isMobile = false,
}: ITooltipProps) {
  return (
    <BlueprintTooltip
      disabled={disabled}
      hoverOpenDelay={openDelay}
      hoverCloseDelay={closeDelay}
      content={
        <div
          className={cn(
            "shadow-md relative z-50 block max-w-xs gap-1 overflow-hidden rounded-md bg-surface-1 p-2 text-11 break-words text-secondary",
            {
              hidden: isMobile,
            },
            className
          )}
        >
          {tooltipHeading && <h5 className="font-medium text-primary">{tooltipHeading}</h5>}
          {tooltipContent}
        </div>
      }
      position={position}
      renderTarget={({ isOpen: _isOpen, ...injectedProps }) => {
        const target = children as ReactElement<TooltipTargetProps>;
        const targetProps = injectedProps as TooltipTargetProps;

        return cloneElement(target, {
          ...targetProps,
          className: cn(target.props.className, targetProps.className),
          ref: mergeRefs(target.props.ref, targetProps.ref),
          onBlur: compose(target.props.onBlur, targetProps.onBlur),
          onContextMenu: compose(target.props.onContextMenu, targetProps.onContextMenu),
          onFocus: compose(target.props.onFocus, targetProps.onFocus),
          onMouseEnter: compose(target.props.onMouseEnter, targetProps.onMouseEnter),
          onMouseLeave: compose(target.props.onMouseLeave, targetProps.onMouseLeave),
        });
      }}
    ></BlueprintTooltip>
  );
}
