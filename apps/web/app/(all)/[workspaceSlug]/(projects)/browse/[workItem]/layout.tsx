/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute, Outlet } from "@tanstack/react-router";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ProjectWorkItemDetailsHeader } from "./header";

function ProjectIssueDetailsLayout(): React.JSX.Element | null {
  return (
    <>
      <ProjectWorkItemDetailsHeader />
      <ContentWrapper className="overflow-hidden">
        <Outlet />
      </ContentWrapper>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/browse/$workItem")({
  component: ProjectIssueDetailsLayout,
});
