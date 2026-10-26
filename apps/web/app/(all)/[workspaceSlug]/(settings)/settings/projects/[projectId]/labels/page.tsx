/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useRef } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";

// components
import { EUserPermissions } from "@plane/constants";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { ProjectSettingsLabelList } from "@/components/labels";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
// hooks
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { LabelsProjectSettingsHeader } from "./header";

function LabelsSettingsPage(): React.JSX.Element | null {
  const { workspaceSlug, projectId } = Route.useParams();
  // store hooks
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const { can: canProject, isPending, isError } = useProjectPermissions(workspaceSlug, projectId);

  const pageTitle = currentProjectDetails?.name ? `${currentProjectDetails?.name} - Labels` : undefined;

  const scrollableContainerRef = useRef<HTMLDivElement | null>(null);

  // derived values
  const canPerformProjectMemberActions = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  // Enable Auto Scroll for Labels list
  useEffect(() => {
    const element = scrollableContainerRef.current;

    if (!element) return;

    return combine(
      autoScrollForElements({
        element,
      })
    );
  }, []);

  if (isPending) return null;

  if (isError || !canPerformProjectMemberActions) {
    return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<LabelsProjectSettingsHeader />}>
      <PageHead title={pageTitle} />
      <div ref={scrollableContainerRef} className="size-full">
        <ProjectSettingsLabelList />
      </div>
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/labels")({
  component: LabelsSettingsPage,
});
