/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useRef, useState } from "react";

import { STICKIES_PER_PAGE } from "@plane/constants";
import { ContentWrapper, Loader } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
import { useStickies } from "@/components/stickies/provider";
import { StickiesLayout } from "./stickies-list";

export const StickiesInfinite = function StickiesInfinite() {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // hooks
  const { list, stickyIds } = useStickies();
  //state
  const [elementRef, setElementRef] = useState<HTMLDivElement | null>(null);

  // ref
  const containerRef = useRef<HTMLDivElement>(null);

  const handleLoadMore = () => {
    if (list.isFetching || !list.hasNextPage) return;
    void list.fetchNextPage();
  };
  const hasNextPage = list.hasNextPage;
  const shouldObserve = hasNextPage && !list.isFetching;
  const workspaceStickies = stickyIds;
  useIntersectionObserver(containerRef, shouldObserve ? elementRef : null, handleLoadMore);

  return (
    <ContentWrapper ref={containerRef} className="space-y-4">
      <StickiesLayout
        workspaceSlug={workspaceSlug.toString()}
        intersectionElement={
          hasNextPage &&
          workspaceStickies?.length >= STICKIES_PER_PAGE && (
            <div
              className={cn("box-border flex min-h-[300px] w-full p-2")}
              ref={setElementRef}
              id="intersection-element"
            >
              <div className="flex min-h-[300px] w-full rounded-sm">
                <Loader className="h-full w-full">
                  <Loader.Item height="100%" width="100%" />
                </Loader>
              </div>
            </div>
          )
        }
      />
    </ContentWrapper>
  );
};
