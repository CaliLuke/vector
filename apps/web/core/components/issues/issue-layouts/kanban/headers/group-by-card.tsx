/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";

// lucide icons
import { Minimize2, Maximize2, Circle } from "lucide-react";
import { PlusIcon } from "@plane/propel/icons";
import type { TIssue, TIssueKanbanFilters, TIssueGroupByOptions } from "@plane/types";
// ui
import { CustomMenu } from "@plane/ui";
// components
import { useGroupWorkItemModals } from "../../group-work-item-modals";
// constants

interface IHeaderGroupByCard {
  sub_group_by: TIssueGroupByOptions | undefined;
  group_by: TIssueGroupByOptions | undefined;
  column_id: string;
  icon?: ReactNode;
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
  const verticalAlignPosition = sub_group_by ? false : collapsedGroups.group_by.includes(column_id);
  const { canAddExisting, openCreate, openExisting, modals } = useGroupWorkItemModals({
    issuePayload,
    isEpic,
    addIssuesToView,
  });

  return (
    <>
      {modals}
      <div
        className={`relative flex flex-shrink-0 gap-1 py-1.5 ${
          verticalAlignPosition ? `w-[44px] flex-col items-center` : `w-full flex-row items-center`
        }`}
      >
        <GroupHeading icon={icon} title={title} count={count} verticalAlignPosition={verticalAlignPosition} />

        {sub_group_by === null && (
          <button
            type="button"
            aria-label={verticalAlignPosition ? "Expand group" : "Collapse group"}
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

        {!disableIssueCreation && (
          <GroupCreationControl canAddExisting={canAddExisting} onCreate={openCreate} onAddExisting={openExisting} />
        )}
      </div>
    </>
  );
};

function GroupCreationControl({
  canAddExisting,
  onCreate,
  onAddExisting,
}: {
  canAddExisting: boolean;
  onCreate: () => void;
  onAddExisting: () => void;
}) {
  const className =
    "flex h-[20px] w-[20px] flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm bg-layer-transparent transition-all hover:bg-layer-transparent-hover";
  if (!canAddExisting) {
    return (
      <button type="button" aria-label="Create work item" className={className} onClick={onCreate}>
        <PlusIcon width={14} strokeWidth={2} />
      </button>
    );
  }
  return (
    <CustomMenu
      customButton={
        <span className={className}>
          <PlusIcon height={14} width={14} strokeWidth={2} />
        </span>
      }
      placement="bottom-end"
    >
      <CustomMenu.MenuItem onClick={onCreate}>
        <span className="flex items-center justify-start gap-2">Create work item</span>
      </CustomMenu.MenuItem>
      <CustomMenu.MenuItem onClick={onAddExisting}>
        <span className="flex items-center justify-start gap-2">Add an existing work item</span>
      </CustomMenu.MenuItem>
    </CustomMenu>
  );
}

function GroupHeading({
  icon,
  title,
  count,
  verticalAlignPosition,
}: Pick<IHeaderGroupByCard, "icon" | "title" | "count"> & { verticalAlignPosition: boolean }) {
  return (
    <>
      {" "}
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
        <div className={`flex-shrink-0 text-13 font-medium text-tertiary ${verticalAlignPosition ? `pr-0.5` : `pl-2`}`}>
          {count || 0}
        </div>
      </div>
    </>
  );
}
