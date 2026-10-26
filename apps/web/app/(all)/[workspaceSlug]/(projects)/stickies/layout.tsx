/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Outlet } from "@tanstack/react-router";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { StickiesProvider } from "@/components/stickies/provider";
import { WorkspaceStickyHeader } from "./header";

function WorkspaceStickiesLayout(): React.JSX.Element | null {
  const { workspaceSlug } = Route.useParams();
  return (
    <StickiesProvider key={workspaceSlug} workspaceSlug={workspaceSlug}>
      <AppHeader header={<WorkspaceStickyHeader />} />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </StickiesProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/stickies")({
  component: WorkspaceStickiesLayout,
});
