/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";

// ui
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { InboxIssueCreateModalRoot } from "@/components/inbox/modals/create-modal";
// hooks
import { useProjectDetail } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { IntakeIcon } from "@plane/propel/icons";

export const ProjectInboxHeader = function ProjectInboxHeader() {
  // states
  const [createIssueModal, setCreateIssueModal] = useState(false);
  // router
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/intake",
  });
  // store hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { t } = useTranslation();

  const { data: currentProjectDetails, isPending: projectPending } = useProjectDetail(workspaceSlug, projectId);

  // derived value
  const isAuthorized = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST]);

  return (
    <Header>
      <Header.LeftItem>
        <div className="flex flex-grow items-center gap-4">
          <Breadcrumbs isLoading={projectPending}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label="Intake"
                  href={`/${workspaceSlug}/projects/${projectId}/intake/`}
                  icon={<IntakeIcon className="h-4 w-4 text-tertiary" />}
                  isLast
                />
              }
              isLast
            />
          </Breadcrumbs>
        </div>
      </Header.LeftItem>
      <Header.RightItem>
        {currentProjectDetails?.inbox_view && workspaceSlug && projectId && isAuthorized ? (
          <div className="flex items-center gap-2">
            <InboxIssueCreateModalRoot
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              modalState={createIssueModal}
              handleModalClose={() => setCreateIssueModal(false)}
            />
            <Button variant="primary" size="lg" onClick={() => setCreateIssueModal(true)}>
              {t("add_work_item")}
            </Button>
          </div>
        ) : (
          <></>
        )}
      </Header.RightItem>
    </Header>
  );
};
