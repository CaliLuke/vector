/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
// components
import type { TPowerKPageType } from "@/components/power-k/core/types";
// hooks
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { projectRosterMemberIds } from "@/helpers/member-rosters";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkItemByIdentifier } from "@/hooks/use-work-item-detail";
// local imports
import { PowerKMembersMenu } from "../../../../menus/members";
import { PowerKWorkItemCyclesMenu } from "./cycles-menu";
import { PowerKWorkItemEstimatesMenu } from "./estimates-menu";
import { PowerKWorkItemLabelsMenu } from "./labels-menu";
import { PowerKWorkItemModulesMenu } from "./modules-menu";
import { PowerKWorkItemPrioritiesMenu } from "./priorities-menu";
import { PowerKProjectStatesMenu } from "./states-menu";

type Props = {
  activePage: TPowerKPageType | null;
  handleSelection: (data: unknown) => void;
};

export const PowerKWorkItemContextBasedPages = function PowerKWorkItemContextBasedPages(props: Props) {
  const { activePage, handleSelection } = props;
  // navigation
  const { workspaceSlug, workItem: entityIdentifier } = useParams({ strict: false });
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const projectCatalogue = useWorkspaceProjectDetails(workspaceSlug?.toString());
  const [projectIdentifier, sequenceId] = (entityIdentifier?.toString() ?? "").split("-");
  const identifierProjectId = projectCatalogue.projects?.find(
    (project) => project.identifier === projectIdentifier
  )?.id;
  const entityDetails = useWorkItemByIdentifier(
    workspaceSlug?.toString(),
    projectIdentifier,
    sequenceId,
    identifierProjectId
  ).issue;
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const { memberships } = useProjectMemberRoster(workspaceSlug, entityDetails?.project_id ?? undefined);
  const projectMemberIds = projectRosterMemberIds(memberships, membersById, currentUser?.id, false);

  if (!entityDetails) return null;

  return (
    <WorkItemContextPage
      activePage={activePage}
      handleSelection={handleSelection}
      workItemDetails={entityDetails}
      workspaceSlug={workspaceSlug}
      projectMemberIds={projectMemberIds ?? undefined}
    />
  );
};

type WorkItemDetails = NonNullable<ReturnType<typeof useWorkItemByIdentifier>["issue"]>;
type WorkItemContextPageProps = Props & {
  workItemDetails: WorkItemDetails;
  workspaceSlug: string | undefined;
  projectMemberIds: string[] | undefined;
};

function WorkItemContextPage({
  activePage,
  handleSelection,
  workItemDetails,
  workspaceSlug,
  projectMemberIds,
}: WorkItemContextPageProps) {
  switch (activePage) {
    case "update-work-item-state":
      return <PowerKProjectStatesMenu handleSelect={handleSelection} workItemDetails={workItemDetails} />;
    case "update-work-item-priority":
      return <PowerKWorkItemPrioritiesMenu handleSelect={handleSelection} workItemDetails={workItemDetails} />;
    case "update-work-item-assignee":
      return (
        <PowerKMembersMenu
          workspaceSlug={workspaceSlug}
          handleSelect={handleSelection}
          userIds={projectMemberIds}
          value={workItemDetails.assignee_ids}
        />
      );
    case "update-work-item-estimate":
      return <PowerKWorkItemEstimatesMenu handleSelect={handleSelection} workItemDetails={workItemDetails} />;
    case "update-work-item-cycle":
      return <PowerKWorkItemCyclesMenu handleSelect={handleSelection} workItemDetails={workItemDetails} />;
    case "update-work-item-module":
      return <PowerKWorkItemModulesMenu handleSelect={handleSelection} workItemDetails={workItemDetails} />;
    case "update-work-item-labels":
      return <PowerKWorkItemLabelsMenu handleSelect={handleSelection} workItemDetails={workItemDetails} />;
    default:
      return null;
  }
}
