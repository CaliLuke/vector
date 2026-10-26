/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useInfiniteQuery } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";
import { StateGroupIcon } from "@plane/propel/icons";
import type { IState, IIssueDisplayProperties } from "@plane/types";
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
import { stateIssuesOptions } from "@/queries/issues";
import type { IssuePage, PublicIssueFilters } from "@/queries/issues";
import { IssueBlock } from "./list/block";
import { KanbanIssueBlock } from "./kanban/block";
import { HeaderGroupByCard as ListHeader } from "./list/headers/group-by-card";
import { HeaderGroupByCard as KanbanHeader } from "./kanban/headers/group-by-card";

const displayProperties: IIssueDisplayProperties = {
  key: true,
  state: true,
  labels: true,
  priority: true,
  due_date: true,
};
type Props = {
  anchor: string;
  state: IState;
  layout: "list" | "kanban";
  filters: PublicIssueFilters;
  firstPage: IssuePage;
  updatedAt: number;
};
export function IssueStateGroup({ anchor, state, layout, filters, firstPage, updatedAt }: Props) {
  const [expanded, setExpanded] = useState(true);
  const container = useRef<HTMLDivElement | null>(null);
  const [sentinel, setSentinel] = useState<HTMLDivElement | null>(null);
  const query = useInfiniteQuery({
    ...stateIssuesOptions(anchor, filters, state.id),
    initialData: { pages: [firstPage], pageParams: ["50:0:0"] },
    initialDataUpdatedAt: updatedAt,
  });
  const { hasNextPage, isFetching, isFetchNextPageError, fetchNextPage } = query;
  const loadMore = useCallback(() => {
    if (hasNextPage && !isFetching && !isFetchNextPageError) void fetchNextPage();
  }, [hasNextPage, isFetching, isFetchNextPageError, fetchNextPage]);
  useIntersectionObserver(container, layout === "kanban" ? sentinel : null, loadMore, "0% 100% 100% 100%");
  const issues = Array.from(
    new Map(query.data.pages.flatMap((page) => page.issues).map((issue) => [issue.id, issue])).values()
  );
  const count = query.data.pages[0].count;
  const icon = <StateGroupIcon stateGroup={state.group} color={state.color} />;
  return (
    <div className={layout === "kanban" ? "flex h-full w-[350px] shrink-0 flex-col" : "relative flex flex-col"}>
      <div className="sticky top-0 z-2 shrink-0 bg-surface-1">
        {layout === "kanban" ? (
          <KanbanHeader groupBy="state" icon={icon} title={state.name} count={count} />
        ) : (
          <ListHeader
            groupID={state.id}
            icon={icon}
            title={state.name}
            count={count}
            toggleListGroup={() => setExpanded((value) => !value)}
          />
        )}
      </div>
      {(layout === "kanban" || expanded) && (
        <div ref={container} className={layout === "kanban" ? "min-h-[120px] overflow-y-auto" : "relative"}>
          {issues.map((issue) =>
            layout === "kanban" ? (
              <KanbanIssueBlock key={issue.id} issue={issue} groupId={state.id} displayProperties={displayProperties} />
            ) : (
              <IssueBlock key={issue.id} issue={issue} displayProperties={displayProperties} />
            )
          )}
          {query.isError && (
            <p role="alert" className="p-3">
              Could not load work items.
            </p>
          )}
          {(query.hasNextPage || query.isError) && (
            <button
              className="w-full p-3 text-13 text-accent-primary"
              disabled={query.isFetching}
              onClick={() => {
                if (query.isFetchNextPageError || query.hasNextPage) void query.fetchNextPage();
                else void query.refetch();
              }}
            >
              {query.isFetching ? "Loading…" : query.isError ? "Retry" : "Load more ↓"}
            </button>
          )}
          {query.hasNextPage && !query.isError && <div ref={setSentinel} />}
        </div>
      )}
    </div>
  );
}
