/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { isTreeDropTargetData, getTreeDropInstruction as getInstructionFromPayload } from "@/lib/tree-drop-instruction";
import { useCallback, useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import type { DragLocationHistory, DropTargetRecord } from "@atlaskit/pragmatic-drag-and-drop/types";
import type { ElementDragPayload } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import orderBy from "lodash-es/orderBy.js";

import { FolderPlus } from "lucide-react";
import { Disclosure, Transition } from "@headlessui/react";
import { IS_FAVORITE_MENU_OPEN } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { ChevronRightIcon } from "@plane/propel/icons";
// ui
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import type { IFavorite } from "@plane/types";
// helpers
import { cn } from "@plane/utils/common";
// hooks
import { useFavorites } from "@/hooks/use-favorites";
import { useLocalStorage } from "@plane/hooks";
// plane web components
import { FavoriteFolder } from "./favorite-folder";
import { FavoriteRoot } from "./favorite-items";
import { NewFavoriteFolder } from "./new-fav-folder";
import { IconButton } from "@plane/propel/icon-button";

export function SidebarFavoritesMenu() {
  // states
  const [createNewFolder, setCreateNewFolder] = useState<boolean | string | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  // navigation
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const {
    groupedFavorites: groupedFavoritesById,
    favorites,
    canAccess,
    isLoading,
    isError,
    retry,
    deleteFavorite,
    removeFromFavoriteFolder,
    reOrderFavorite,
    moveFavoriteToFolder,
  } = useFavorites(workspaceSlug);
  const groupedFavorites = groupedFavoritesById ?? {};
  // translation
  const { t } = useTranslation();
  // local storage
  const { setValue: toggleFavoriteMenu, storedValue } = useLocalStorage<boolean>(IS_FAVORITE_MENU_OPEN, false);
  // derived values
  const isFavoriteMenuOpen = !!storedValue;
  // refs
  const containerRef = useRef<HTMLDivElement>(null);
  const elementRef = useRef<HTMLDivElement>(null);

  const handleMoveToFolder = (sourceId: string, destinationId: string) => {
    moveFavoriteToFolder(sourceId, destinationId).catch(() => {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("error"),
        message: t("failed_to_move_favorite"),
      });
    });
  };

  const handleDrop = (self: DropTargetRecord, source: ElementDragPayload, location: DragLocationHistory) => {
    const isFolder = self.data?.isGroup;
    const dropTargets = location?.current?.dropTargets ?? [];
    if (!dropTargets || dropTargets.length <= 0) return;
    const dropTarget =
      dropTargets.length > 1 ? dropTargets.find((target: DropTargetRecord) => target?.data?.isChild) : dropTargets[0];

    const dropTargetData = dropTarget?.data;

    if (!dropTarget || !isTreeDropTargetData(dropTargetData)) return;
    const instruction = getInstructionFromPayload(dropTarget, source, location);
    const parentId = instruction === "make-child" ? dropTargetData.id : dropTargetData.parentId;
    const droppedFavId = instruction !== "make-child" ? dropTargetData.id : undefined;
    const sourceData = source.data;

    if (!isTreeDropTargetData(sourceData)) return;
    if (isFolder) {
      // handle move to a new parent folder if dropped on a folder
      if (parentId && parentId !== sourceData.parentId) {
        handleMoveToFolder(sourceData.id, parentId); /**parent id  */
      }
      // handle reordering at root level
      if (droppedFavId) {
        if (instruction != "make-child") {
          handleReorder(sourceData.id, droppedFavId, instruction); /** sequence */
        }
      }
    } else {
      //handling reordering for favorites
      if (droppedFavId) {
        handleReorder(sourceData.id, droppedFavId, instruction); /** sequence */
      }
    }

    /**remove if dropped outside and source is a child */
    if (!parentId && sourceData.isChild) {
      handleRemoveFromFavoritesFolder(sourceData.id); /**parent null */
    }
  };

  const handleRemoveFromFavorites = (favorite: IFavorite) => {
    deleteFavorite(favorite.id)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("success"),
          message: t("favorite_removed_successfully"),
        });
        return;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("error"),
          message: t("something_went_wrong"),
        });
      });
  };

  const handleRemoveFromFavoritesFolder = (favoriteId: string) => {
    removeFromFavoriteFolder(favoriteId).catch(() => {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("error"),
        message: t("failed_to_move_favorite"),
      });
    });
  };

  const handleReorder = useCallback(
    (favoriteId: string, droppedFavId: string, edge: string | undefined) => {
      reOrderFavorite(favoriteId, droppedFavId, edge).catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("error"),
          message: t("failed_to_reorder_favorite"),
        });
      });
    },
    [reOrderFavorite, t]
  );

  useEffect(() => {
    const element = elementRef.current;

    if (!element) return;

    return combine(
      dropTargetForElements({
        element,
        onDragEnter: () => {
          setIsDragging(true);
        },
        onDragLeave: () => {
          setIsDragging(false);
        },
        onDragStart: () => {
          setIsDragging(true);
        },
        onDrop: ({ source }) => {
          setIsDragging(false);
          if (!isTreeDropTargetData(source.data)) return;
          const sourceId = source.data.id;
          if (!groupedFavorites[sourceId]?.parent) return;
        },
      })
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [elementRef.current, isDragging]);

  return (
    <>
      <Disclosure as="div" defaultOpen ref={containerRef}>
        <div
          ref={elementRef}
          className={cn(
            "group/favorites-button flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-placeholder hover:bg-layer-transparent-hover"
          )}
        >
          <Disclosure.Button
            as="button"
            type="button"
            className={cn(
              "flex w-full items-center gap-1 text-left text-13 font-semibold whitespace-nowrap text-placeholder",
              {
                "bg-layer-1 opacity-60": isDragging,
              }
            )}
            onClick={() => toggleFavoriteMenu(!isFavoriteMenuOpen)}
            aria-label={t(
              isFavoriteMenuOpen
                ? "aria_labels.projects_sidebar.close_favorites_menu"
                : "aria_labels.projects_sidebar.open_favorites_menu"
            )}
          >
            <span className="text-13 font-semibold">{t("favorites")}</span>
          </Disclosure.Button>
          <div className="pointer-events-none flex items-center opacity-0 group-hover/favorites-button:pointer-events-auto group-hover/favorites-button:opacity-100">
            <Tooltip tooltipHeading={t("create_folder")} tooltipContent="">
              <IconButton
                variant="ghost"
                size="sm"
                disabled={!canAccess}
                onClick={() => {
                  setCreateNewFolder(true);
                  if (!isFavoriteMenuOpen) toggleFavoriteMenu(!isFavoriteMenuOpen);
                }}
                aria-label={t("aria_labels.projects_sidebar.create_favorites_folder")}
                icon={FolderPlus}
              />
            </Tooltip>
            <Disclosure.Button
              as="button"
              type="button"
              className="grid flex-shrink-0 place-items-center rounded-sm p-0.5 hover:bg-layer-transparent-hover"
              onClick={() => toggleFavoriteMenu(!isFavoriteMenuOpen)}
              aria-label={t(
                isFavoriteMenuOpen
                  ? "aria_labels.projects_sidebar.close_favorites_menu"
                  : "aria_labels.projects_sidebar.open_favorites_menu"
              )}
            >
              <ChevronRightIcon
                className={cn("size-3 flex-shrink-0 transition-all", {
                  "rotate-90": isFavoriteMenuOpen,
                })}
              />
            </Disclosure.Button>
          </div>
        </div>
        <Transition
          show={isFavoriteMenuOpen}
          enter="transition duration-100 ease-out"
          enterFrom="transform scale-95 opacity-0"
          enterTo="transform scale-100 opacity-100"
          leave="transition duration-75 ease-out"
          leaveFrom="transform scale-100 opacity-100"
          leaveTo="transform scale-95 opacity-0"
        >
          <Disclosure.Panel as="div" className="mt-0.5 flex flex-col gap-0.5" static>
            {createNewFolder && canAccess && (
              <NewFavoriteFolder setCreateNewFolder={setCreateNewFolder} actionType="create" />
            )}
            {isError ? (
              <div className="px-8 py-1.5 text-11 text-danger-primary" role="alert">
                <span>{t("something_went_wrong")}</span>
                <button type="button" className="ml-2 underline" onClick={() => void retry()}>
                  {t("retry")}
                </button>
              </div>
            ) : isLoading ? (
              <span className="px-8 py-1.5 text-11 font-medium text-placeholder">{t("loading")}</span>
            ) : !canAccess || favorites === undefined || Object.keys(groupedFavorites).length === 0 ? (
              <>
                <span className="px-8 py-1.5 text-11 font-medium text-placeholder">{t("no_favorites_yet")}</span>
              </>
            ) : (
              orderBy(Object.values(groupedFavorites), "sequence", "desc")
                .filter((fav) => !fav.parent)
                .map((fav, index, { length }) => (
                  <>
                    {fav?.is_folder ? (
                      <FavoriteFolder
                        favorite={fav}
                        isLastChild={index === length - 1}
                        handleRemoveFromFavorites={handleRemoveFromFavorites}
                        handleRemoveFromFavoritesFolder={handleRemoveFromFavoritesFolder}
                        handleDrop={handleDrop}
                      />
                    ) : (
                      <FavoriteRoot
                        workspaceSlug={workspaceSlug.toString()}
                        favorite={fav}
                        isLastChild={index === length - 1}
                        parentId={undefined}
                        handleRemoveFromFavorites={handleRemoveFromFavorites}
                        handleDrop={handleDrop}
                      />
                    )}
                  </>
                ))
            )}
          </Disclosure.Panel>
        </Transition>
      </Disclosure>
    </>
  );
}
