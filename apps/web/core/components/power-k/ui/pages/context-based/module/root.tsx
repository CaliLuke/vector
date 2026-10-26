/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// components
import type { TPowerKPageType } from "@/components/power-k/core/types";
import { PowerKMembersMenu } from "@/components/power-k/menus/members";
// hooks
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { projectRosterMemberIds } from "@/helpers/member-rosters";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useModuleDetails } from "@/hooks/use-modules";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
// local imports
import { PowerKModuleStatusMenu } from "./status-menu";

type Props = {
  activePage: TPowerKPageType | null;
  handleSelection: (data: unknown) => void;
};

export const PowerKModuleContextBasedPages = function PowerKModuleContextBasedPages(props: Props) {
  const { activePage, handleSelection } = props;
  // navigation
  const { workspaceSlug, projectId, moduleId } = useParams({ strict: false });
  // store hooks
  const detail = useModuleDetails(workspaceSlug, projectId, moduleId);
  const { data: currentUser } = useCurrentUser();
  // derived values
  const moduleDetails = detail.module;
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const { memberships } = useProjectMemberRoster(workspaceSlug, moduleDetails?.project_id);
  const projectMemberIds = projectRosterMemberIds(memberships, membersById, currentUser?.id, false);

  if (detail.isError || detail.isAccessDenied) return <CatalogueReadFailure label="modules" retry={detail.retry} />;
  if (!moduleDetails) return null;

  return (
    <>
      {/* members menu */}
      {activePage === "update-module-member" && moduleDetails && (
        <PowerKMembersMenu
          workspaceSlug={workspaceSlug}
          handleSelect={handleSelection}
          userIds={projectMemberIds ?? undefined}
          value={moduleDetails.member_ids}
        />
      )}
      {/* status menu */}
      {activePage === "update-module-status" && moduleDetails?.status && (
        <PowerKModuleStatusMenu handleSelect={handleSelection} value={moduleDetails.status} />
      )}
    </>
  );
};
