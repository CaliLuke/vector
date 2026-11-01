/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Dispatch, ReactElement, SetStateAction } from "react";
import React, { useCallback, useEffect, useEffectEvent, useState, useRef } from "react";
// helpers
import { usePlatformOS } from "@plane/hooks";
import { cn } from "@plane/utils/common";

interface ResizableSidebarProps {
  showPeek?: boolean;
  togglePeek: (value?: boolean) => void;
  isCollapsed?: boolean;
  width: number;
  setWidth: Dispatch<SetStateAction<number>>;
  defaultWidth?: number;
  minWidth?: number;
  maxWidth?: number;
  defaultCollapsed?: boolean;
  peekDuration?: number;
  toggleCollapsed: (value?: boolean) => void;
  onWidthChange?: (width: number) => void;
  onCollapsedChange?: (collapsed: boolean) => void;
  className?: string;
  children?: ReactElement;
  extendedSidebar?: ReactElement;
  isAnyExtendedSidebarExpanded?: boolean;
  isAnySidebarDropdownOpen?: boolean;
}

export function ResizableSidebar({
  showPeek = false,
  togglePeek,
  peekDuration = 500,
  isCollapsed = false,
  toggleCollapsed: toggleCollapsedProp,
  onCollapsedChange,
  width,
  setWidth,
  onWidthChange,
  minWidth = 236,
  maxWidth = 350,
  className = "",
  children,
  extendedSidebar,
  isAnyExtendedSidebarExpanded = false,
  isAnySidebarDropdownOpen = false,
}: ResizableSidebarProps) {
  // states
  const [isResizing, setIsResizing] = useState(false);
  // refs
  const peekTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const initialWidthRef = useRef<number>(0);
  const initialMouseXRef = useRef<number>(0);
  // hooks
  const { isMobile } = usePlatformOS();
  // handlers
  const setShowPeek = useCallback(
    (value: boolean) => {
      togglePeek(value);
    },
    [togglePeek]
  );

  const handleResize = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;

      const deltaX = e.clientX - initialMouseXRef.current;
      const newWidth = Math.min(Math.max(initialWidthRef.current + deltaX, minWidth), maxWidth);
      setWidth(newWidth);
    },
    [isResizing, minWidth, maxWidth, setWidth]
  );

  const startResizing = useCallback(
    (e: React.MouseEvent) => {
      setIsResizing(true);
      initialWidthRef.current = width;
      initialMouseXRef.current = e.clientX;
    },
    [width]
  );

  const stopResizing = useCallback(() => {
    setIsResizing(false);
  }, []);

  const toggleCollapsed = useCallback(() => {
    toggleCollapsedProp();
    setShowPeek(false);
    if (peekTimeoutRef.current) {
      clearTimeout(peekTimeoutRef.current);
    }
  }, [toggleCollapsedProp, setShowPeek]);

  const closePeek = useEffectEvent(() => {
    if (isCollapsed && !isAnyExtendedSidebarExpanded && !isAnySidebarDropdownOpen) setShowPeek(false);
  });
  const handlePeekEnter = () => clearTimeout(peekTimeoutRef.current);
  const handlePeekLeave = () => {
    clearTimeout(peekTimeoutRef.current);
    peekTimeoutRef.current = setTimeout(closePeek, peekDuration);
  };

  useEffect(() => {
    if (!isResizing) return undefined;
    const { cursor, userSelect } = document.body.style;
    document.addEventListener("mousemove", handleResize);
    document.addEventListener("mouseup", stopResizing);
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    return () => {
      document.removeEventListener("mousemove", handleResize);
      document.removeEventListener("mouseup", stopResizing);
      document.body.style.cursor = cursor;
      document.body.style.userSelect = userSelect;
    };
  }, [isResizing, handleResize, stopResizing]);

  // Clean up timeout on unmount
  useEffect(
    () => () => {
      if (peekTimeoutRef.current) {
        clearTimeout(peekTimeoutRef.current);
      }
    },
    []
  );

  const resetExpandedPeek = useEffectEvent(() => {
    setShowPeek(false);
    clearTimeout(peekTimeoutRef.current);
  });
  useEffect(() => {
    if (!isCollapsed) resetExpandedPeek();
  }, [isCollapsed]);

  // Call external handlers when state changes
  useEffect(() => {
    onWidthChange?.(width);
  }, [width, onWidthChange]);

  useEffect(() => {
    onCollapsedChange?.(isCollapsed);
  }, [isCollapsed, onCollapsedChange]);

  const resizeHandle = (
    <SidebarResizeHandle
      isResizing={isResizing}
      onStart={startResizing}
      onToggle={toggleCollapsed}
      width={width}
      setWidth={setWidth}
      minWidth={minWidth}
      maxWidth={maxWidth}
    />
  );
  return (
    <SidebarPanels
      width={width}
      isResizing={isResizing}
      isCollapsed={isCollapsed}
      isMobile={isMobile}
      showPeek={showPeek}
      className={className}
      isAnyExtendedSidebarExpanded={isAnyExtendedSidebarExpanded}
      onPeekEnter={handlePeekEnter}
      onPeekLeave={handlePeekLeave}
      resizeHandle={resizeHandle}
      extendedSidebar={extendedSidebar}
    >
      {children}
    </SidebarPanels>
  );
}

type SidebarPanelsProps = Pick<ResizableSidebarProps, "children" | "extendedSidebar" | "width"> & {
  isResizing: boolean;
  isCollapsed: boolean;
  isMobile: boolean;
  showPeek: boolean;
  className: string;
  isAnyExtendedSidebarExpanded: boolean;
  onPeekEnter: () => void;
  onPeekLeave: () => void;
  resizeHandle: ReactElement;
};

function SidebarPanels({
  width,
  isResizing,
  isCollapsed,
  isMobile,
  showPeek,
  className,
  isAnyExtendedSidebarExpanded,
  onPeekEnter,
  onPeekLeave,
  resizeHandle,
  children,
  extendedSidebar,
}: SidebarPanelsProps) {
  const mainWidth = isCollapsed ? 0 : width;
  const peekVisible = isCollapsed && showPeek;
  return (
    <>
      {/* Main Sidebar */}
      <aside
        id="main-sidebar"
        className={cn(
          "z-20 h-full border-r border-subtle bg-surface-1",
          !isResizing && "transition-all duration-300 ease-in-out",
          isCollapsed ? "w-0 translate-x-[-100%] opacity-0" : "translate-x-0 opacity-100",
          isMobile && "absolute",
          className
        )}
        style={{
          width: `${mainWidth}px`,
          minWidth: `${mainWidth}px`,
          maxWidth: `${mainWidth}px`,
        }}
        aria-label="Main sidebar"
        data-prevent-outside-click={isMobile}
      >
        <div
          className={cn(
            "group/sidebar relative flex h-full w-full flex-col overflow-hidden bg-surface-1 pt-3",
            isAnyExtendedSidebarExpanded && "rounded-none"
          )}
        >
          {children}

          {resizeHandle}
        </div>
      </aside>
      {/* Peek View */}
      <aside
        className={cn(
          "shadow-sm absolute left-0 z-20 h-full bg-surface-1",
          !isResizing && "transition-all duration-300 ease-in-out",
          peekVisible ? "translate-x-0 opacity-100" : "translate-x-[-100%] opacity-0",
          "pointer-events-none",
          peekVisible && "pointer-events-auto",
          !showPeek ? "w-0" : "w-full"
        )}
        style={{
          width: `${width}px`,
        }}
        aria-label="Sidebar peek view"
      >
        <div
          onPointerEnter={onPeekEnter}
          onPointerLeave={onPeekLeave}
          onFocus={onPeekEnter}
          onBlur={onPeekLeave}
          className={cn(
            "group/sidebar relative z-20 flex h-full w-full flex-col overflow-hidden bg-surface-1 pt-4",
            "self-center rounded-md rounded-tl-none rounded-bl-none border-r border-subtle",
            isAnyExtendedSidebarExpanded && "rounded-none"
          )}
        >
          {children}
          {resizeHandle}
        </div>
      </aside>

      {/* Extended Sidebar */}
      {extendedSidebar}
    </>
  );
}

function SidebarResizeHandle({
  isResizing,
  onStart,
  onToggle,
  width,
  setWidth,
  minWidth,
  maxWidth,
}: {
  isResizing: boolean;
  onStart: (event: React.MouseEvent) => void;
  onToggle: () => void;
  width: number;
  setWidth: Dispatch<SetStateAction<number>>;
  minWidth: number;
  maxWidth: number;
}) {
  return (
    <button
      type="button"
      aria-label="Resize sidebar"
      title="Drag or use arrow keys to resize; Enter to collapse"
      className={cn(
        "absolute top-0 right-0 z-[20] h-full w-1 cursor-ew-resize transition-all duration-200",
        isResizing ? "bg-layer-1" : "hover:bg-surface-2"
      )}
      onDoubleClick={onToggle}
      onMouseDown={onStart}
      onKeyDown={(event) => {
        if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
          event.preventDefault();
          setWidth(Math.min(maxWidth, Math.max(minWidth, width + (event.key === "ArrowRight" ? 10 : -10))));
        } else if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onToggle();
        }
      }}
    />
  );
}
