/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
// components
import { ViewListLoader } from "@/components/ui/loader/view-list-loader";
// hooks
import { getWorkspaceViewsLoadState, useWorkspaceViews } from "@/hooks/use-workspace-views";
// local imports
import { GlobalViewListItem } from "./view-list-item";

type Props = {
  searchQuery: string;
};

export function GlobalViewsList(props: Props) {
  const { searchQuery } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const viewsQuery = useWorkspaceViews(workspaceSlug);
  const loadState = getWorkspaceViewsLoadState(viewsQuery);
  if (loadState.isError) {
    return (
      <div role="alert" className="p-4 text-13 text-secondary">
        <p>Workspace views could not be loaded.</p>
        <button type="button" className="mt-2 text-accent-primary" onClick={() => void viewsQuery.retry()}>
          Retry
        </button>
      </div>
    );
  }
  if (loadState.isLoading) return <ViewListLoader />;
  if (!viewsQuery.canRead)
    return <output className="block p-4 text-13 text-secondary">Workspace views are unavailable.</output>;
  if (viewsQuery.isPending || !workspaceSlug) return <ViewListLoader />;

  const filteredViews = (viewsQuery.views ?? []).filter((view) =>
    view.name?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <>
      {filteredViews.map((view) => (
        <GlobalViewListItem key={view.id} view={view} />
      ))}
    </>
  );
}
