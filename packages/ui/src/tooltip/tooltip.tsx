/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { cloneElement, useState, type HTMLProps, type MouseEvent, type ReactElement, type ReactNode } from "react";
import {
  arrow,
  autoPlacement,
  autoUpdate,
  flip,
  FloatingArrow,
  FloatingPortal,
  offset,
  shift,
  useDismiss,
  useFloating,
  useFocus,
  useHover,
  useInteractions,
  useMergeRefs,
  useRole,
} from "@floating-ui/react";
import { cn } from "../utils/classname";

const positions = {
  top: "top",
  right: "right",
  bottom: "bottom",
  left: "left",
  auto: undefined,
  "auto-end": undefined,
  "auto-start": undefined,
  "bottom-left": "bottom-start",
  "bottom-right": "bottom-end",
  "left-bottom": "left-end",
  "left-top": "left-start",
  "right-bottom": "right-end",
  "right-top": "right-start",
  "top-left": "top-start",
  "top-right": "top-end",
} as const;

export type TPosition = keyof typeof positions;

interface ITooltipProps {
  tooltipHeading?: string;
  tooltipContent: string | ReactNode;
  position?: TPosition;
  children: ReactElement<HTMLProps<HTMLElement>>;
  disabled?: boolean;
  className?: string;
  openDelay?: number;
  closeDelay?: number;
  isMobile?: boolean;
  /** @deprecated The overlay is lazy; target handlers always mount for keyboard access. */
  renderByDefault?: boolean;
}

export function Tooltip({
  tooltipHeading,
  tooltipContent,
  position = "top",
  children,
  disabled = false,
  className = "",
  openDelay = 200,
  closeDelay = 0,
  isMobile = false,
}: ITooltipProps) {
  const [isOpen, setIsOpen] = useState(false);
  if (disabled && isOpen) setIsOpen(false);
  const [reference, setReference] = useState<Element | null>(null);
  const [floating, setFloating] = useState<HTMLElement | null>(null);
  const [arrowElement, setArrowElement] = useState<SVGSVGElement | null>(null);
  const placement = positions[position];
  const { floatingStyles, context } = useFloating({
    open: isOpen && !disabled,
    onOpenChange: setIsOpen,
    elements: { reference, floating },
    placement,
    whileElementsMounted: autoUpdate,
    middleware: [offset(11), placement ? flip() : autoPlacement(), shift(), arrow({ element: arrowElement })],
  });
  const hover = useHover(context, {
    enabled: !disabled,
    delay: { open: openDelay, close: closeDelay },
    mouseOnly: true,
  });
  const focus = useFocus(context, { enabled: !disabled, visibleOnly: false });
  const dismiss = useDismiss(context);
  const role = useRole(context, { role: "tooltip" });
  const { getReferenceProps, getFloatingProps } = useInteractions([hover, focus, dismiss, role]);
  const ref = useMergeRefs([setReference, children.props.ref]);

  return (
    <>
      {cloneElement(children, {
        ...getReferenceProps({
          ...children.props,
          onContextMenu: (event: MouseEvent<HTMLElement>) => {
            children.props.onContextMenu?.(event);
            if (event.defaultPrevented) setIsOpen(false);
          },
        }),
        ref,
      })}
      {isOpen && !disabled && (
        <FloatingPortal>
          <div
            ref={setFloating}
            style={floatingStyles}
            className={cn("z-50", { hidden: isMobile })}
            data-placement={context.placement}
            {...getFloatingProps()}
          >
            <FloatingArrow ref={setArrowElement} context={context} width={22} height={7} className="fill-surface-1" />
            <div
              className={cn(
                "shadow-md relative z-50 block max-w-xs gap-1 overflow-hidden rounded-md bg-surface-1 p-2 text-11 break-words text-secondary",
                className
              )}
            >
              {tooltipHeading && <h5 className="font-medium text-primary">{tooltipHeading}</h5>}
              {tooltipContent}
            </div>
          </div>
        </FloatingPortal>
      )}
    </>
  );
}
