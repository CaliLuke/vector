/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useSubmenuRegistry } from "@plane/propel/menu";
import React, { useEffect, useMemo, useRef, useState } from "react";
import ReactDOM from "react-dom";
// hooks
import { usePlatformOS } from "../../hooks/use-platform-os";
// helpers
import { cn } from "../../utils";
// components
import { getDropdownEventTarget } from "../helper";
export { DropdownPortal as Portal } from "../portal";
import { ContextMenuItem } from "./item";

export type TContextMenuItem = {
  key: string;
  customContent?: React.ReactNode;
  title?: string;
  description?: string;
  icon?: React.ComponentType<{ className?: string }>;
  action: () => void;
  shouldRender?: boolean;
  closeOnClick?: boolean;
  disabled?: boolean;
  className?: string;
  iconClassName?: string;
  nestedMenuItems?: TContextMenuItem[];
};

// Context for managing nested menus
export const ContextMenuContext = React.createContext<{
  closeAllSubmenus: () => void;
  registerSubmenu: (closeSubmenu: () => void) => () => void;
  portalContainer?: Element | null;
} | null>(null);

type ContextMenuProps = {
  parentRef: React.RefObject<HTMLElement | null>;
  items: TContextMenuItem[];
  portalContainer?: Element | null;
};

function ContextMenuWithoutPortal(props: ContextMenuProps) {
  const { parentRef, items, portalContainer } = props;
  // states
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState({
    x: 0,
    y: 0,
  });
  const [activeItemIndex, setActiveItemIndex] = useState<number>(0);
  // refs
  const contextMenuRef = useRef<HTMLDivElement>(null);
  const { closeAllSubmenus, registerSubmenu } = useSubmenuRegistry();
  // derived values
  const renderedItems = useMemo(() => items.filter((item) => item.shouldRender !== false), [items]);
  const { isMobile } = usePlatformOS();

  const handleClose = React.useCallback(() => {
    closeAllSubmenus();
    setIsOpen(false);
    setActiveItemIndex(0);
  }, [closeAllSubmenus]);

  // calculate position of context menu
  useEffect(() => {
    const parentElement = parentRef.current;
    const contextMenu = contextMenuRef.current;
    if (!parentElement || !contextMenu) return undefined;

    const handleContextMenu = (e: MouseEvent) => {
      if (isMobile) return;

      e.preventDefault();
      e.stopPropagation();

      const contextMenuWidth = contextMenu.clientWidth;
      const contextMenuHeight = contextMenu.clientHeight;

      const clickX = e?.pageX || 0;
      const clickY = e?.pageY || 0;

      // check if there's enough space at the bottom, otherwise show at the top
      let top = clickY;
      if (clickY + contextMenuHeight > window.innerHeight) top = clickY - contextMenuHeight;

      // check if there's enough space on the right, otherwise show on the left
      let left = clickX;
      if (clickX + contextMenuWidth > window.innerWidth) left = clickX - contextMenuWidth;

      setPosition({ x: left, y: top });
      setIsOpen(true);
    };

    const hideContextMenu = (e: KeyboardEvent) => {
      if (isOpen && e.key === "Escape") handleClose();
    };

    parentElement.addEventListener("contextmenu", handleContextMenu);
    window.addEventListener("keydown", hideContextMenu);

    return () => {
      parentElement.removeEventListener("contextmenu", handleContextMenu);
      window.removeEventListener("keydown", hideContextMenu);
    };
  }, [isMobile, isOpen, parentRef, handleClose]);

  // handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen || !renderedItems.length || e.defaultPrevented) return;

      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveItemIndex((prev) => (prev + 1) % renderedItems.length);
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveItemIndex((prev) => (prev - 1 + renderedItems.length) % renderedItems.length);
      }
      if (e.key === "Enter") {
        e.preventDefault();
        const item = renderedItems[activeItemIndex];
        if (item && !item.disabled) {
          renderedItems[activeItemIndex].action();
          if (item.closeOnClick !== false) handleClose();
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [activeItemIndex, isOpen, renderedItems, handleClose]);

  // Custom handler for nested menu portal clicks
  React.useEffect(() => {
    const handleDocumentClick = (event: MouseEvent) => {
      const target = getDropdownEventTarget(event);
      if (!target) return;

      // Check if the click is on a nested menu element
      const isNestedMenuClick = target.closest('[data-context-submenu="true"]');
      const isMainMenuClick = contextMenuRef.current?.contains(target);

      // Also check if the target itself has the data attribute
      const isNestedMenuElement = target.hasAttribute("data-context-submenu");

      // If it's a nested menu click, main menu click, or nested menu element, don't close
      if (isNestedMenuClick || isMainMenuClick || isNestedMenuElement) {
        return;
      }

      // If menu is open and it's an outside click, close it
      if (isOpen) {
        handleClose();
      }
    };

    if (isOpen) {
      // Use capture phase to ensure we handle the event before other handlers
      document.addEventListener("mousedown", handleDocumentClick, true);
      return () => {
        document.removeEventListener("mousedown", handleDocumentClick, true);
      };
    }

    return undefined;
  }, [isOpen, handleClose]);

  const contextValue = useMemo(
    () => ({ closeAllSubmenus, registerSubmenu, portalContainer }),
    [closeAllSubmenus, registerSubmenu, portalContainer]
  );

  return (
    <div
      className={cn(
        "pointer-events-none fixed top-0 left-0 z-30 h-screen w-screen cursor-default opacity-0 transition-opacity",
        {
          "pointer-events-auto opacity-100": isOpen,
        }
      )}
    >
      <div
        ref={contextMenuRef}
        className="vertical-scrollbar fixed scrollbar-sm max-h-72 min-w-[12rem] overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 px-2 py-2.5"
        style={{
          top: position.y,
          left: position.x,
        }}
        data-context-menu="true"
      >
        <ContextMenuContext.Provider value={contextValue}>
          {renderedItems.map((item, index) => (
            <ContextMenuItem
              key={item.key}
              handleActiveItem={() => setActiveItemIndex(index)}
              handleClose={handleClose}
              isActive={index === activeItemIndex}
              item={item}
            />
          ))}
        </ContextMenuContext.Provider>
      </div>
    </div>
  );
}

export function ContextMenu(props: ContextMenuProps) {
  let contextMenu = <ContextMenuWithoutPortal {...props} />;
  const portal = document.querySelector("#context-menu-portal");
  if (portal) contextMenu = ReactDOM.createPortal(contextMenu, portal);
  return contextMenu;
}
