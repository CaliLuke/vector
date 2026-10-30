/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { Header, Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
// components
import { AppHeader } from "@/components/core/app-header";
import { TabNavigationRoot } from "@/components/navigation";
import { AppSidebarToggleButton } from "@/components/sidebar/sidebar-toggle-button";
// hooks
import { useSidebarCollapsed } from "@/providers/sidebar-state";
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkItemByIdentifier } from "@/hooks/use-work-item-detail";
// local components
import { WorkItemDetailsHeader } from "./work-item-header";

export const ProjectWorkItemDetailsHeader = function ProjectWorkItemDetailsHeader() {
  // router
  const { workspaceSlug, workItem } = useParams({ from: "/_authenticated/$workspaceSlug/_projects/browse/$workItem" });
  // store hooks
  const sidebarCollapsed = useSidebarCollapsed();
  const projectCatalogue = useWorkspaceProjectDetails();
  const [projectIdentifier, sequenceId] = (workItem ?? "").split("-");
  const projectId = projectCatalogue.projects?.find((project) => project.identifier === projectIdentifier)?.id;
  const detail = useWorkItemByIdentifier(workspaceSlug, projectIdentifier, sequenceId, projectId);
  const issueDetails = detail.issue;
  // preferences
  const { preferences: projectPreferences } = useProjectNavigationPreferences();

  return (
    <>
      {projectPreferences.navigationMode === "TABBED" && (
        <div className="z-20">
          <Row className="flex h-header w-full items-center gap-2 border-b border-subtle bg-surface-1">
            <div className="flex h-full w-full items-center gap-2 divide-x divide-subtle">
              <div className="flex size-full flex-1 items-center gap-2">
                {sidebarCollapsed && (
                  <div className="shrink-0">
                    <AppSidebarToggleButton />
                  </div>
                )}
                <Header className={cn("h-full", { "pl-1.5": !sidebarCollapsed })}>
                  <Header.LeftItem className="h-full max-w-full">
                    <TabNavigationRoot
                      workspaceSlug={workspaceSlug}
                      projectId={issueDetails?.project_id?.toString() ?? ""}
                    />
                  </Header.LeftItem>
                </Header>
              </div>
            </div>
          </Row>
        </div>
      )}
      <AppHeader header={<WorkItemDetailsHeader />} />
    </>
  );
};
