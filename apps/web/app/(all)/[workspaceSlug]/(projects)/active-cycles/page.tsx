/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Link } from "@tanstack/react-router";

import { getButtonStyling } from "@plane/propel/button";
// components
import { PageHead } from "@/components/core/page-title";
// hooks
import { useCurrentWorkspace } from "@/hooks/use-workspaces";

function WorkspaceActiveCyclesPage(): React.JSX.Element | null {
  const { data: currentWorkspace } = useCurrentWorkspace();
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace.name} - Active Cycles` : undefined;

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="flex flex-col items-start gap-4 p-6">
        <h1 className="text-20 font-semibold">Active cycles</h1>
        <p className="text-14 text-secondary">
          The workspace-wide cycle overview is not available in this build. You can view and manage cycles within each
          project.
        </p>
        {currentWorkspace && (
          <Link
            to="/$workspaceSlug/projects/"
            params={{ workspaceSlug: currentWorkspace.slug }}
            className={getButtonStyling("primary", "base")}
          >
            View projects
          </Link>
        )}
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/active-cycles/")({
  component: WorkspaceActiveCyclesPage,
});
