/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// ui
import { Button } from "@plane/propel/button";
import { DraftIcon } from "@plane/propel/icons";
import { EIssuesStoreType } from "@plane/types";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { CountChip } from "@/components/common/count-chip";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";

// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useParams } from "@tanstack/react-router";
import { useWorkspaceDraftList } from "@/hooks/use-workspace-drafts";

export function WorkspaceDraftHeader() {
  // state
  const [isDraftIssueModalOpen, setIsDraftIssueModalOpen] = useState(false);
  // store hooks
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { can } = useWorkspacePermissions(workspaceSlug);
  const { paginationInfo } = useWorkspaceDraftList(workspaceSlug);
  const { t } = useTranslation();
  // check if user is authorized to create draft work item
  const isAuthorizedUser = can([EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST]);

  return (
    <>
      <CreateUpdateIssueModal
        isOpen={isDraftIssueModalOpen}
        storeType={EIssuesStoreType.WORKSPACE_DRAFT}
        onClose={() => setIsDraftIssueModalOpen(false)}
        isDraft
      />
      <Header>
        <Header.LeftItem>
          <div className="flex items-center gap-2.5">
            <Breadcrumbs>
              <Breadcrumbs.Item
                component={
                  <BreadcrumbLink label={t("drafts")} icon={<DraftIcon className="h-4 w-4 text-tertiary" />} />
                }
              />
            </Breadcrumbs>
            {paginationInfo?.total_count && paginationInfo?.total_count > 0 ? (
              <CountChip count={paginationInfo?.total_count} />
            ) : (
              <></>
            )}
          </div>
        </Header.LeftItem>

        <Header.RightItem>
          <Button
            variant="primary"
            size="lg"
            className="items-center gap-1"
            onClick={() => setIsDraftIssueModalOpen(true)}
            disabled={!isAuthorizedUser}
          >
            {t("workspace_draft_issues.draft_an_issue")}
          </Button>
        </Header.RightItem>
      </Header>
    </>
  );
}
