/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams, useLocation } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { treeDropPositionTarget } from "@/helpers/tree-drop-position";

import { Pin, PinOff } from "lucide-react";
// plane imports
import type { IWorkspaceSidebarNavigationItem } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Tooltip } from "@plane/propel/tooltip";
import { DropIndicator } from "@plane/ui";
import { cn } from "@plane/utils/common";
// components
import { SidebarNavItem } from "@/components/sidebar/sidebar-navigation";
// hooks
import { useSidebarActions } from "@/providers/sidebar-state";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceNavigationPreferences } from "@/hooks/use-navigation-preferences";
// local imports
import { getSidebarNavigationItemIcon } from "./helper";
import { SidebarDragHandle } from "./drag-handle";

type TExtendedSidebarItemProps = {
  item: IWorkspaceSidebarNavigationItem;
  handleOnNavigationItemDrop?: (
    sourceId: string | undefined,
    destinationId: string | undefined,
    shouldDropAtEnd: boolean
  ) => void;
  disableDrag?: boolean;
  disableDrop?: boolean;
  isLastChild: boolean;
};

export function ExtendedSidebarItem(props: TExtendedSidebarItemProps) {
  const { item, handleOnNavigationItemDrop, disableDrag = false, disableDrop = false, isLastChild } = props;
  const { t } = useTranslation();
  // states
  const [isDragging, setIsDragging] = useState(false);
  const [instruction, setInstruction] = useState<"DRAG_OVER" | "DRAG_BELOW" | undefined>(undefined);
  // refs
  const navigationIemRef = useRef<HTMLDivElement | null>(null);
  const dragHandleRef = useRef<HTMLButtonElement | null>(null);

  // nextjs hooks
  const pathname = useLocation({ select: (location) => location.pathname });
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const { toggleExtendedSidebar } = useSidebarActions();
  const { data } = useCurrentUser();
  const { can } = useWorkspacePermissions(workspaceSlug);
  const { preferences: workspacePreferences, toggleWorkspaceItem } = useWorkspaceNavigationPreferences();

  // derived values
  const isPinned = workspacePreferences.items[item.key]?.is_pinned ?? false;

  const handleLinkClick = () => toggleExtendedSidebar(true);

  useEffect(() => {
    const element = navigationIemRef.current;
    const dragHandleElement = dragHandleRef.current;

    if (!element) return;

    return combine(
      draggable({
        element,
        canDrag: () => !disableDrag,
        dragHandle: dragHandleElement ?? undefined,
        getInitialData: () => ({ id: item.key, dragInstanceId: "NAVIGATION" }), // var1
        onDragStart: () => {
          setIsDragging(true);
        },
        onDrop: () => {
          setIsDragging(false);
        },
      }),
      dropTargetForElements({
        element,
        canDrop: ({ source }) =>
          !disableDrop && source?.data?.id !== item.key && source?.data?.dragInstanceId === "NAVIGATION",
        ...treeDropPositionTarget(item.key, isLastChild, setInstruction, handleOnNavigationItemDrop),
      })
    );
  }, [isLastChild, handleOnNavigationItemDrop, disableDrag, disableDrop, item.key]);

  const itemHref =
    item.key === "your_work"
      ? `/${workspaceSlug.toString()}${item.href}${data?.id}`
      : `/${workspaceSlug.toString()}${item.href}`;
  const isActive = itemHref === pathname;

  const pinNavigationItem = (key: string) => {
    toggleWorkspaceItem(key, true);
  };

  const unPinNavigationItem = (key: string) => {
    toggleWorkspaceItem(key, false);
  };

  const icon = getSidebarNavigationItemIcon(item.key);

  if (!can(item.access)) {
    return null;
  }

  return (
    <div
      id={`sidebar-${item.key}`}
      className={cn("relative", {
        "bg-layer-1 opacity-60": isDragging,
      })}
      ref={navigationIemRef}
    >
      <DropIndicator classNames="absolute top-0" isVisible={instruction === "DRAG_OVER"} />
      <div
        className={cn(
          "group/project-item relative flex w-full items-center rounded-md text-primary hover:bg-surface-2"
        )}
        id={`${item.key}`}
      >
        {!disableDrag && (
          <SidebarDragHandle
            // isMobile={isMobile}
            tooltipContent={t("drag_to_rearrange")}
            position="top-start"
            disabled={isDragging}
            handleClassName={cn(
              "absolute top-1/2 -left-3 flex -translate-y-1/2 cursor-grab items-center justify-center rounded text-placeholder opacity-0 group-hover/project-item:opacity-100",
              {
                "cursor-grabbing": isDragging,
                "opacity-100": isDragging,
              }
            )}
            ref={dragHandleRef}
          />
        )}
        <SidebarNavItem isActive={isActive}>
          <Link to={itemHref} onClick={() => handleLinkClick()} className="group flex-grow">
            <div className="flex items-center gap-1.5 py-[1px]">
              {icon}
              <p className="text-13 leading-5 font-medium">{t(item.labelTranslationKey)}</p>
            </div>
          </Link>
          <div className="flex items-center gap-2">
            {isPinned ? (
              <Tooltip tooltipContent="Unpin">
                <PinOff
                  className="size-3.5 flex-shrink-0 text-placeholder outline-none hover:text-tertiary"
                  onClick={() => unPinNavigationItem(item.key)}
                />
              </Tooltip>
            ) : (
              <Tooltip tooltipContent="Pin">
                <Pin
                  className="size-3.5 flex-shrink-0 text-placeholder outline-none hover:text-tertiary"
                  onClick={() => pinNavigationItem(item.key)}
                />
              </Tooltip>
            )}
          </div>
        </SidebarNavItem>
      </div>
      {isLastChild && <DropIndicator isVisible={instruction === "DRAG_BELOW"} />}
    </div>
  );
}
