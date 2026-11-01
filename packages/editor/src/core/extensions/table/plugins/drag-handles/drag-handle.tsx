/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  autoUpdate,
  flip,
  FloatingOverlay,
  FloatingFocusManager,
  FloatingPortal,
  shift,
  useClick,
  useDismiss,
  useFloating,
  useInteractions,
  useRole,
} from "@floating-ui/react";
import type { Editor } from "@tiptap/core";
import { Ellipsis } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { ComponentType, MouseEvent as ReactMouseEvent } from "react";
import { cn } from "@plane/utils/common";
import { CORE_EXTENSIONS } from "@/constants/extension";

type TableDragHandleProps = {
  editor: Editor;
  axis: "row" | "column";
  Dropdown: ComponentType<{ editor: Editor; onClose: () => void }>;
  startDrag: (event: MouseEvent) => (() => void) | undefined;
};

export function TableDragHandle({ editor, axis, Dropdown, startDrag }: TableDragHandleProps) {
  const [open, setOpen] = useState(false);
  const cancelDrag = useRef<(() => void) | undefined>(undefined);
  useEffect(() => () => cancelDrag.current?.(), []);

  useEffect(() => {
    if (!open) return undefined;
    editor.commands.addActiveDropbarExtension(CORE_EXTENSIONS.TABLE);
    return () => {
      editor.commands.removeActiveDropbarExtension(CORE_EXTENSIONS.TABLE);
    };
  }, [editor, open]);

  const { refs, floatingStyles, context } = useFloating({
    placement: "bottom-start",
    middleware: [
      flip({ fallbackPlacements: ["top-start", "bottom-start", "top-end", "bottom-end"] }),
      shift({ padding: 8 }),
    ],
    open,
    onOpenChange: setOpen,
    whileElementsMounted: autoUpdate,
  });
  const click = useClick(context);
  const dismiss = useDismiss(context);
  const role = useRole(context);
  const { getReferenceProps, getFloatingProps } = useInteractions([dismiss, click, role]);

  const handleMouseDown = (event: ReactMouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    event.preventDefault();
    cancelDrag.current?.();
    cancelDrag.current = startDrag(event.nativeEvent);
  };
  const isRow = axis === "row";
  return (
    <>
      <div
        className={cn(
          "absolute top-0 left-0 z-20 flex items-center justify-center",
          isRow
            ? "table-row-handle-container h-full -translate-x-1/2"
            : "table-col-handle-container w-full -translate-y-1/2"
        )}
      >
        <button
          ref={(node) => refs.setReference(node)}
          {...getReferenceProps()}
          type="button"
          onMouseDown={handleMouseDown}
          aria-label={isRow ? "Row options" : "Column options"}
          className={cn(
            "rounded-sm border border-strong-1 bg-layer-1 transition-all duration-200 outline-none",
            isRow ? "py-1" : "px-1",
            open ? "border-accent-strong bg-accent-primary !opacity-100" : "hover:bg-layer-1-hover"
          )}
        >
          <Ellipsis className={cn("size-4 text-primary", { "rotate-90": isRow })} />
        </button>
      </div>
      {open && (
        <FloatingPortal>
          <FloatingOverlay style={{ zIndex: 99 }} lockScroll />
          <FloatingFocusManager context={context}>
            <div
              className="max-h-[90vh] w-[12rem] overflow-y-auto rounded-md border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 shadow-raised-200"
              ref={(node) => refs.setFloating(node)}
              {...getFloatingProps()}
              style={{ ...floatingStyles, zIndex: 100 }}
            >
              <Dropdown editor={editor} onClose={() => setOpen(false)} />
            </div>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}
