/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { treeDropHandlers } from "@/lib/tree-drop-instruction";
import { useDraggablePreview } from "@/hooks/use-draggable-preview";
import { useEffect, useRef, useState } from "react";
import type { DropTargetRecord, DragLocationHistory } from "@atlaskit/pragmatic-drag-and-drop/types";
import type { ElementDragPayload } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { attachInstruction } from "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item";

// plane imports
import { useOutsideClickDetector } from "@plane/hooks";
import type { IFavorite, InstructionType } from "@plane/types";
import { DropIndicator } from "@plane/ui";
// hooks
import { useFavoriteItemDetails } from "@/hooks/use-favorite-item-details";
// local imports
import { getCanDrop } from "../favorites.helpers";
import { FavoriteItemDragHandle, FavoriteItemQuickAction, FavoriteItemTitle, FavoriteItemWrapper } from "./common";

type Props = {
  isLastChild: boolean;
  parentId: string | undefined;
  workspaceSlug: string;
  favorite: IFavorite;
  handleRemoveFromFavorites: (favorite: IFavorite) => void;
  handleDrop: (self: DropTargetRecord, source: ElementDragPayload, location: DragLocationHistory) => void;
};

export const FavoriteRoot = function FavoriteRoot(props: Props) {
  // props
  const { isLastChild, parentId, workspaceSlug, favorite, handleRemoveFromFavorites, handleDrop } = props;
  // store hooks
  const { itemLink, itemIcon, itemTitle } = useFavoriteItemDetails(workspaceSlug, favorite);
  //state
  const [isDragging, setIsDragging] = useState(false);
  const [isMenuActive, setIsMenuActive] = useState(false);
  const [instruction, setInstruction] = useState<InstructionType | undefined>(undefined);

  //ref
  const elementRef = useRef<HTMLDivElement>(null);
  const actionSectionRef = useRef<HTMLDivElement | null>(null);

  const handleQuickAction = (value: boolean) => setIsMenuActive(value);

  useDraggablePreview({
    elementRef,
    getInitialData: () => ({ id: favorite.id, isGroup: false, isChild: !!parentId, parentId: parentId ?? null }),
    preview: <FavoriteItemTitle href={itemLink} icon={itemIcon} title={itemTitle} />,
    onDraggingChange: setIsDragging,
  });

  // drag and drop
  useEffect(() => {
    const rootElement = elementRef.current;

    if (!rootElement) return undefined;
    const initialData = { id: favorite.id, isGroup: false, isChild: !!parentId, parentId: parentId ?? null };
    return dropTargetForElements({
      element: rootElement,
      canDrop: ({ source }) => getCanDrop(source, favorite, !!parentId),
      onDragStart: () => {
        setIsDragging(true);
      },
      getData: ({ input, element }) => {
        const blockedStates: InstructionType[] = ["make-child"];
        if (!isLastChild) {
          blockedStates.push("reorder-below");
        }

        return attachInstruction(initialData, {
          input,
          element,
          currentLevel: 1,
          indentPerLevel: 0,
          mode: isLastChild ? "last-in-group" : "standard",
          block: blockedStates,
        });
      },
      ...treeDropHandlers(setInstruction, handleDrop),
    });
  }, [handleDrop, parentId, isLastChild, favorite]);

  useOutsideClickDetector(actionSectionRef, () => setIsMenuActive(false));

  return (
    <>
      {isDragging && <DropIndicator isVisible={instruction === "reorder-above"} />}
      <FavoriteItemWrapper elementRef={elementRef} isMenuActive={isMenuActive}>
        <FavoriteItemDragHandle isDragging={isDragging} sort_order={favorite.sort_order} />
        <FavoriteItemTitle href={itemLink} icon={itemIcon} title={itemTitle} />
        <FavoriteItemQuickAction
          favorite={favorite}
          ref={actionSectionRef}
          isMenuActive={isMenuActive}
          onChange={handleQuickAction}
          handleRemoveFromFavorites={handleRemoveFromFavorites}
        />
      </FavoriteItemWrapper>
      {isLastChild && <DropIndicator isVisible={instruction === "reorder-below"} />}
    </>
  );
};
