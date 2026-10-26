/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useRef, useState } from "react";
import omit from "lodash-es/omit.js";
import { SquareStackIcon } from "lucide-react";
import { CopyIcon, EditIcon, TrashIcon } from "@plane/propel/icons";
// plane utils
import { Tooltip } from "@plane/propel/tooltip";
import type { TWorkspaceDraftIssue } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import type { TContextMenuItem } from "@plane/ui";
import { Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
// hooks
import { useSidebarCollapsed } from "@/providers/sidebar-state";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkspaceDraftMutations } from "@/hooks/use-workspace-drafts";
// local imports
import { IdentifierText } from "../issue-detail/identifier-text";
import { CreateUpdateIssueModal } from "../issue-modal/modal";
import { WorkspaceDraftIssueDeleteIssueModal } from "./delete-modal";
import { DraftIssueProperties } from "./draft-issue-properties";
import { WorkspaceDraftIssueQuickActions } from "./quick-action";

type Props = {
  workspaceSlug: string;
  issue: TWorkspaceDraftIssue;
};

export function DraftIssueBlock(props: Props) {
  // props
  const { workspaceSlug, issue } = props;
  // states
  const [moveToIssue, setMoveToIssue] = useState(false);
  const [createUpdateIssueModal, setCreateUpdateIssueModal] = useState(false);
  const [issueToEdit, setIssueToEdit] = useState<TWorkspaceDraftIssue | undefined>(undefined);
  const [deleteIssueModal, setDeleteIssueModal] = useState(false);
  // hooks
  const { updateDraft, deleteDraft } = useWorkspaceDraftMutations(workspaceSlug);
  const isSidebarCollapsed = useSidebarCollapsed();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  // ref
  const issueRef = useRef<HTMLDivElement | null>(null);
  // derived values
  const projectIdentifier =
    (issue &&
      issue.project_id &&
      (projectRows?.find((project) => project.id === issue.project_id)?.identifier ?? "")) ||
    undefined;
  const duplicateIssuePayload = omit(
    {
      ...issue,
      name: `${issue.name} (copy)`,
      is_draft: true,
      sourceIssueId: issue.id,
    },
    ["id"]
  );

  const MENU_ITEMS: TContextMenuItem[] = [
    {
      key: "edit",
      title: "edit",
      icon: EditIcon,
      action: () => {
        setIssueToEdit(issue);
        setCreateUpdateIssueModal(true);
      },
    },
    {
      key: "make-a-copy",
      title: "make_a_copy",
      icon: CopyIcon,
      action: () => {
        setCreateUpdateIssueModal(true);
      },
    },
    {
      key: "move-to-issues",
      title: "move_to_project",
      icon: SquareStackIcon,
      action: () => {
        setMoveToIssue(true);
        setIssueToEdit(issue);
        setCreateUpdateIssueModal(true);
      },
    },
    {
      key: "delete",
      title: "delete",
      icon: TrashIcon,
      action: () => {
        setDeleteIssueModal(true);
      },
    },
  ];

  return (
    <>
      <WorkspaceDraftIssueDeleteIssueModal
        data={issue}
        isOpen={deleteIssueModal}
        handleClose={() => setDeleteIssueModal(false)}
        onSubmit={async () => deleteDraft(issue.id)}
      />
      <CreateUpdateIssueModal
        isOpen={createUpdateIssueModal}
        onClose={() => {
          setCreateUpdateIssueModal(false);
          setIssueToEdit(undefined);
          setMoveToIssue(false);
        }}
        data={issueToEdit ?? duplicateIssuePayload}
        storeType={EIssuesStoreType.WORKSPACE_DRAFT}
        fetchIssueDetails={false}
        moveToIssue={moveToIssue}
        isDraft
      />
      <div
        id={`issue-${issue.id}`}
        className="relative w-full cursor-pointer border-b border-b-subtle-1"
        onDoubleClick={() => {
          setIssueToEdit(issue);
          setCreateUpdateIssueModal(true);
        }}
      >
        <Row
          ref={issueRef}
          className={cn(
            "group/list-block relative flex min-h-11 flex-col gap-3 bg-layer-transparent py-3 text-13 transition-colors hover:bg-layer-transparent-hover",
            {
              "md:flex-row md:items-center": isSidebarCollapsed,
              "lg:flex-row lg:items-center": !isSidebarCollapsed,
            }
          )}
        >
          <div className="flex w-full truncate">
            <div className="flex flex-grow items-center gap-0.5 truncate">
              <div className="flex items-center gap-1">
                <div className="flex-shrink-0">
                  {projectIdentifier && (
                    <div className="flex items-center space-x-2">
                      <IdentifierText
                        identifier={projectIdentifier}
                        enableClickToCopyIdentifier
                        size="xs"
                        variant="tertiary"
                      />
                    </div>
                  )}
                </div>

                {/* sub-issues chevron */}
                <div className="grid size-4 flex-shrink-0 place-items-center" />
              </div>

              <Tooltip tooltipContent={issue.name} position="top-start" renderByDefault={false}>
                <p className="w-full cursor-pointer truncate text-13 text-primary">{issue.name || "Untitled"}</p>
              </Tooltip>
            </div>

            {/* quick actions */}
            <div
              className={cn("block rounded-sm border border-strong", {
                "md:hidden": isSidebarCollapsed,
                "lg:hidden": !isSidebarCollapsed,
              })}
            >
              <WorkspaceDraftIssueQuickActions parentRef={issueRef} MENU_ITEMS={MENU_ITEMS} />
            </div>
          </div>

          <div className="flex flex-shrink-0 items-center gap-2">
            <DraftIssueProperties
              className={`relative flex flex-wrap ${isSidebarCollapsed ? "md:flex-shrink-0 md:flex-grow" : "lg:flex-shrink-0 lg:flex-grow"} items-center gap-2 whitespace-nowrap`}
              issue={issue}
              updateIssue={async (projectId, issueId, data) => {
                await updateDraft(issue.id, data);
              }}
            />
            <div
              className={cn("hidden", {
                "md:flex": isSidebarCollapsed,
                "lg:flex": !isSidebarCollapsed,
              })}
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
              }}
            >
              <WorkspaceDraftIssueQuickActions parentRef={issueRef} MENU_ITEMS={MENU_ITEMS} />
            </div>
          </div>
        </Row>
      </div>
    </>
  );
}
