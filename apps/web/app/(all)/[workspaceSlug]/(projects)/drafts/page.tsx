/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute } from "@tanstack/react-router";
import { PageHead } from "@/components/core/page-title";
import { WorkspaceDraftIssuesRoot } from "@/components/issues/workspace-draft";

function WorkspaceDraftPage(): React.JSX.Element | null {
  const { workspaceSlug } = Route.useParams();
  const pageTitle = "Workspace Draft";

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="relative h-full w-full overflow-hidden overflow-y-auto">
        <WorkspaceDraftIssuesRoot workspaceSlug={workspaceSlug} />
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/drafts/")({
  component: WorkspaceDraftPage,
});
