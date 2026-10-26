/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation } from "@tanstack/react-router";
import { getTreeDropInstruction as getInstructionFromPayload } from "@/lib/tree-drop-instruction";
import { useEffect, useRef, useState } from "react";
import type Masonry from "masonry-layout";
import type { DropTargetRecord, DragLocationHistory } from "@atlaskit/pragmatic-drag-and-drop/types";
import type { ElementDragPayload } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";

import { useTheme } from "next-themes";

// plane imports

import { useTranslation } from "@plane/i18n";
import { PlusIcon } from "@plane/propel/icons";
import { EUserWorkspaceRoles } from "@plane/types";
// assets
import darkStickiesAsset from "@/app/assets/empty-state/stickies/stickies-dark.webp?url";
import lightStickiesAsset from "@/app/assets/empty-state/stickies/stickies-light.webp?url";
import darkStickiesSearchAsset from "@/app/assets/empty-state/stickies/stickies-search-dark.webp?url";
import lightStickiesSearchAsset from "@/app/assets/empty-state/stickies/stickies-search-light.webp?url";
// components
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";
import { StickiesEmptyState } from "@/components/home/widgets/empty-states/stickies";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useStickies } from "@/components/stickies/provider";
// local imports
import { useStickyOperations } from "../sticky/use-operations";
import { StickiesLoader } from "./stickies-loader";
import { StickyDNDWrapper } from "./sticky-dnd-wrapper";

type TStickiesLayout = {
  workspaceSlug: string;
  intersectionElement?: React.ReactNode | null;
};

type TProps = TStickiesLayout & {
  columnCount: number;
};

export const StickiesList = function StickiesList(props: TProps) {
  const { workspaceSlug, intersectionElement, columnCount } = props;
  // navigation
  const pathname = useLocation({ select: (location) => location.pathname });
  // theme hook
  const { resolvedTheme } = useTheme();
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { list, stickyIds: workspaceStickyIds, searchQuery } = useStickies();
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  // sticky operations
  const { stickyOperations } = useStickyOperations({ workspaceSlug: workspaceSlug?.toString() });
  // derived values
  const itemWidth = `${100 / columnCount}%`;
  const totalRows = Math.ceil(workspaceStickyIds.length / columnCount);
  const isStickiesPage = pathname?.includes("stickies");
  const hasGuestLevelPermissions = canWorkspace([
    EUserWorkspaceRoles.ADMIN,
    EUserWorkspaceRoles.MEMBER,
    EUserWorkspaceRoles.GUEST,
  ]);
  const stickiesResolvedPath = resolvedTheme === "light" ? lightStickiesAsset : darkStickiesAsset;
  const stickiesSearchResolvedPath = resolvedTheme === "light" ? lightStickiesSearchAsset : darkStickiesSearchAsset;
  const masonryRef = useRef<MasonryInstance | null>(null);
  const masonryContainerRef = useRef<HTMLDivElement | null>(null);
  const masonryResizeObserverRef = useRef<ResizeObserver | null>(null);
  const shouldRenderMasonry = !list.isPending && workspaceStickyIds.length > 0;

  useEffect(() => {
    if (!shouldRenderMasonry) return;

    const container = masonryContainerRef.current;
    if (!container) return;

    let cancelled = false;
    let masonry: MasonryInstance | null = null;
    let resizeObserver: ResizeObserver | null = null;
    let childObserver: MutationObserver | null = null;

    void import("masonry-layout").then(({ default: Masonry }) => {
      if (cancelled) return;

      const masonryInstance = new Masonry(container, {
        itemSelector: ".sticky-masonry-item",
      });
      if (!masonryInstance.layout || !masonryInstance.reloadItems || !masonryInstance.destroy) {
        throw new Error("The installed masonry-layout package is missing required lifecycle methods.");
      }
      masonry = masonryInstance as MasonryInstance;
      masonryRef.current = masonry;

      resizeObserver = new ResizeObserver(() => masonry?.layout());
      masonryResizeObserverRef.current = resizeObserver;
      container.querySelectorAll<HTMLElement>(".sticky-masonry-item").forEach((item) => resizeObserver?.observe(item));
      childObserver = new MutationObserver((records) => {
        for (const record of records) {
          record.addedNodes.forEach((node) => {
            if (node instanceof HTMLElement && node.matches(".sticky-masonry-item")) resizeObserver?.observe(node);
          });
        }
        masonry?.reloadItems();
        masonry?.layout();
      });
      childObserver.observe(container, { childList: true });
      masonry.layout();
    });

    return () => {
      cancelled = true;
      childObserver?.disconnect();
      resizeObserver?.disconnect();
      masonryResizeObserverRef.current = null;
      masonry?.destroy();
      if (masonryRef.current === masonry) masonryRef.current = null;
    };
  }, [shouldRenderMasonry]);

  const handleLayout = () => {
    masonryRef.current?.layout();
  };

  // Function to determine if an item is in first or last row
  const getRowPositions = (index: number) => {
    const currentRow = Math.floor(index / columnCount);
    return {
      isInFirstRow: currentRow === 0,
      isInLastRow: currentRow === totalRows - 1 || index >= workspaceStickyIds.length - columnCount,
    };
  };

  const handleDrop = (self: DropTargetRecord, source: ElementDragPayload, location: DragLocationHistory) => {
    const dropTargets = location?.current?.dropTargets ?? [];
    if (!dropTargets || dropTargets.length <= 0) return;

    const dropTarget = dropTargets[0];
    if (!dropTarget?.data?.id || !source.data?.id) return;

    const instruction = getInstructionFromPayload(dropTarget, source, location);
    const droppedId = dropTarget.data.id;
    const sourceId = source.data.id;

    try {
      if (!instruction || !droppedId || !sourceId) return;
      stickyOperations.updatePosition(workspaceSlug, sourceId as string, droppedId as string, instruction);
    } catch (error) {
      console.error("Error reordering sticky:", error);
    }
  };

  if (list.isPending) {
    return <StickiesLoader />;
  }

  if (list.isError)
    return (
      <div role="alert">
        Unable to load stickies.{" "}
        <button
          type="button"
          onClick={() => {
            void list.refetch();
          }}
        >
          Try again
        </button>
      </div>
    );

  if (workspaceStickyIds.length === 0) {
    return (
      <div className="grid size-full place-items-center">
        {isStickiesPage ? (
          <>
            {searchQuery ? (
              <SimpleEmptyState
                title={t("stickies.empty_state.search.title")}
                description={t("stickies.empty_state.search.description")}
                assetPath={stickiesSearchResolvedPath}
              />
            ) : (
              <DetailedEmptyState
                title={t("stickies.empty_state.general.title")}
                description={t("stickies.empty_state.general.description")}
                assetPath={stickiesResolvedPath}
                primaryButton={{
                  prependIcon: <PlusIcon className="size-4" />,
                  text: t("stickies.empty_state.general.primary_button.text"),
                  onClick: () => {
                    stickyOperations.create();
                  },
                  disabled: !hasGuestLevelPermissions,
                }}
              />
            )}
          </>
        ) : (
          <StickiesEmptyState />
        )}
      </div>
    );
  }

  return (
    <div className="transition-opacity duration-300 ease-in-out">
      <div ref={masonryContainerRef}>
        {workspaceStickyIds.map((stickyId, index) => {
          const { isInFirstRow, isInLastRow } = getRowPositions(index);
          return (
            <StickyDNDWrapper
              key={stickyId}
              className="sticky-masonry-item"
              stickyId={stickyId}
              workspaceSlug={workspaceSlug.toString()}
              itemWidth={itemWidth}
              handleDrop={handleDrop}
              isLastChild={index === workspaceStickyIds.length - 1}
              isInFirstRow={isInFirstRow}
              isInLastRow={isInLastRow}
              handleLayout={handleLayout}
            />
          );
        })}
        {intersectionElement && (
          <div className="sticky-masonry-item" style={{ width: itemWidth }}>
            {intersectionElement}
          </div>
        )}
      </div>
    </div>
  );
};

type MasonryInstance = Required<Pick<Masonry, "layout" | "reloadItems" | "destroy">>;

export function StickiesLayout(props: TStickiesLayout) {
  // states
  const [containerWidth, setContainerWidth] = useState<number | null>(null);
  // refs
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref?.current) return;

    setContainerWidth(ref?.current.offsetWidth);

    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerWidth(entry.contentRect.width);
      }
    });

    resizeObserver.observe(ref?.current);
    return () => resizeObserver.disconnect();
  }, []);

  const getColumnCount = (width: number | null): number => {
    if (width === null) return 4;

    if (width < 640) return 2; // sm
    if (width < 850) return 3; // md
    if (width < 1024) return 4; // lg
    if (width < 1280) return 5; // xl
    return 6; // 2xl and above
  };
  const columnCount = getColumnCount(containerWidth);

  return (
    <div ref={ref} className="size-full">
      <StickiesList {...props} columnCount={columnCount} />
    </div>
  );
}
