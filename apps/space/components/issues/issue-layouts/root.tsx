/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useQuery } from "@tanstack/react-query";
import { LogoSpinner } from "@/components/common/logo-spinner";
import { IssueAppliedFilters } from "@/components/issues/filters/applied-filters/root";
import { IssuePeekOverview } from "@/components/issues/peek-overview";
import { usePublicIssueView } from "@/hooks/use-public-issue-view";
import { publicIssuesOptions, firstStatePage } from "@/queries/issues";
import { statesOptions } from "@/queries/lookups";
import { IssueStateGroup } from "./state-group";

type Props = { anchor: string; peekId: string | undefined };
export function IssuesLayoutsRoot({ anchor, peekId }: Props) {
  const { layout, filters } = usePublicIssueView(anchor);
  const issues = useQuery(publicIssuesOptions(anchor, filters));
  const states = useRecordCollection(statesOptions(anchor));
  const retry = () => {
    void issues.refetch();
    void states.refetch();
  };
  const error = issues.isError || states.isError;
  return (
    <div className="relative flex size-full flex-col overflow-hidden">
      {peekId && <IssuePeekOverview key={`${anchor}:${peekId}`} anchor={anchor} peekId={peekId} />}
      <IssueAppliedFilters anchor={anchor} />
      {error ? (
        <div role="alert" className="p-5">
          Could not load work items. <button onClick={retry}>Retry</button>
        </div>
      ) : !issues.data || !states.data ? (
        <LogoSpinner />
      ) : issues.data.total_count === 0 ? (
        <div className="grid size-full place-items-center text-secondary">No work items found</div>
      ) : (
        <div className={layout === "kanban" ? "flex h-full gap-2 overflow-x-auto p-5" : "size-full overflow-y-auto"}>
          {states.data
            .filter((state) => !filters.state.length || filters.state.includes(state.id))
            .map((state) => (
              <IssueStateGroup
                key={state.id}
                anchor={anchor}
                state={state}
                layout={layout === "kanban" ? "kanban" : "list"}
                filters={filters}
                firstPage={firstStatePage(issues.data, state.id)}
                updatedAt={issues.dataUpdatedAt}
              />
            ))}
        </div>
      )}
    </div>
  );
}
