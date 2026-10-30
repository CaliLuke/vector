/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

// lucide icons
import { Minimize2, Maximize2, Circle } from "lucide-react";
import { PlusIcon } from "@plane/propel/icons";
import type { TIssue, ISearchIssueResponse, TIssueKanbanFilters, TIssueGroupByOptions } from "@plane/types";
// ui
import { CustomMenu } from "@plane/ui";
// components
import { addSelectedIssuesToView } from "../../add-selected-issues";
import { ExistingIssuesListModal } from "@/components/core/modals/existing-issues-list-modal";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
// constants
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { CreateUpdateEpicModal } from "@/components/epic-modal";

interface IHeaderGroupByCard {
  sub_group_by: TIssueGroupByOptions | undefined;
  group_by: TIssueGroupByOptions | undefined;
  column_id: string;
  icon?: React.ReactNode;
  title: string;
  count: number;
  collapsedGroups: TIssueKanbanFilters;
  handleCollapsedGroups: (toggle: "group_by" | "sub_group_by", value: string) => void;
  issuePayload: Partial<TIssue>;
  disableIssueCreation?: boolean;
  addIssuesToView?: (issueIds: string[]) => Promise<void | TIssue>;
  isEpic?: boolean;
}

export const HeaderGroupByCard = function HeaderGroupByCard(props: IHeaderGroupByCard) {
  const {
    sub_group_by,
    column_id,
    icon,
    title,
    count,
    collapsedGroups,
    handleCollapsedGroups,
    issuePayload,
    disableIssueCreation,
    addIssuesToView,
    isEpic = false,
  } = props;
  const verticalAlignPosition = sub_group_by ? false : collapsedGroups?.group_by.includes(column_id);
  // states
  const [isOpen, setIsOpen] = React.useState(false);
  const [openExistingIssueListModal, setOpenExistingIssueListModal] = React.useState(false);
  // hooks
  const storeType = useIssueListType();
  // router
  const { workspaceSlug, projectId, moduleId, cycleId } = useParams({ strict: false });

  const renderExistingIssueModal = moduleId || cycleId;
  const ExistingIssuesListModalPayload = moduleId ? { module: moduleId } : { cycle: true };

  const handleAddIssuesToView = async (data: ISearchIssueResponse[]) => {
    if (!workspaceSlug || !projectId) return;

    await addSelectedIssuesToView(data, addIssuesToView);
  };

  return (
    <>
      {isEpic ? (
        <CreateUpdateEpicModal isOpen={isOpen} onClose={() => setIsOpen(false)} data={issuePayload} />
      ) : (
        <CreateUpdateIssueModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          data={issuePayload}
          storeType={storeType}
        />
      )}

      {renderExistingIssueModal && (
        <ExistingIssuesListModal
          workspaceSlug={workspaceSlug?.toString()}
          projectId={projectId?.toString()}
          isOpen={openExistingIssueListModal}
          handleClose={() => setOpenExistingIssueListModal(false)}
          searchParams={ExistingIssuesListModalPayload}
          handleOnSubmit={handleAddIssuesToView}
        />
      )}
      <div
        className={`relative flex flex-shrink-0 gap-1 py-1.5 ${
          verticalAlignPosition ? `w-[44px] flex-col items-center` : `w-full flex-row items-center`
        }`}
      >
        <div className="flex size-5 flex-shrink-0 items-center justify-center overflow-hidden rounded-xs">
          {icon ? icon : <Circle width={14} strokeWidth={2} />}
        </div>

        <div
          className={`relative flex gap-1 ${
            verticalAlignPosition ? `flex-col items-center` : `w-full flex-row items-baseline overflow-hidden`
          }`}
        >
          <div
            className={`line-clamp-1 inline-block truncate overflow-hidden font-medium text-primary ${
              verticalAlignPosition ? `max-h-[400px] vertical-lr` : ``
            }`}
          >
            {title}
          </div>
          <div
            className={`flex-shrink-0 text-13 font-medium text-tertiary ${verticalAlignPosition ? `pr-0.5` : `pl-2`}`}
          >
            {count || 0}
          </div>
        </div>

        {sub_group_by === null && (
          <button
            className="flex h-[20px] w-[20px] flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm bg-layer-transparent transition-all hover:bg-layer-transparent-hover"
            onClick={() => handleCollapsedGroups("group_by", column_id)}
          >
            {verticalAlignPosition ? (
              <Maximize2 width={14} strokeWidth={2} />
            ) : (
              <Minimize2 width={14} strokeWidth={2} />
            )}
          </button>
        )}

        {!disableIssueCreation &&
          (renderExistingIssueModal ? (
            <CustomMenu
              customButton={
                <span className="flex h-[20px] w-[20px] flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm bg-layer-transparent transition-all hover:bg-layer-transparent-hover">
                  <PlusIcon height={14} width={14} strokeWidth={2} />
                </span>
              }
              placement="bottom-end"
            >
              <CustomMenu.MenuItem
                onClick={() => {
                  setIsOpen(true);
                }}
              >
                <span className="flex items-center justify-start gap-2">Create work item</span>
              </CustomMenu.MenuItem>
              <CustomMenu.MenuItem
                onClick={() => {
                  setOpenExistingIssueListModal(true);
                }}
              >
                <span className="flex items-center justify-start gap-2">Add an existing work item</span>
              </CustomMenu.MenuItem>
            </CustomMenu>
          ) : (
            <button
              className="flex h-[20px] w-[20px] flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm bg-layer-transparent transition-all hover:bg-layer-transparent-hover"
              onClick={() => {
                setIsOpen(true);
              }}
            >
              <PlusIcon width={14} strokeWidth={2} />
            </button>
          ))}
      </div>
    </>
  );
};
