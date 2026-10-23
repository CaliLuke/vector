/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute, Outlet } from "@tanstack/react-router";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { CyclesListHeader } from "./header";
import { CyclesListMobileHeader } from "./mobile-header";

function ProjectCyclesListLayout(): React.JSX.Element | null {
  return (
    <>
      <AppHeader header={<CyclesListHeader />} mobileHeader={<CyclesListMobileHeader />} />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/cycles")({
  component: ProjectCyclesListLayout,
});
