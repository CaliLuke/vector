import type { IssueQuickAddCallback } from "./types";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useState } from "react";
import type { FC } from "react";
import { useParams } from "@tanstack/react-router";
import { useTranslation } from "@plane/i18n";
import { PlusIcon } from "@plane/propel/icons";
import type { TIssue, TIssueGroupByOptions } from "@plane/types";
import { EIssueLayoutTypes } from "@plane/types";
import { canQuickAddToGroup } from "./grouping";
import { ListQuickAddIssueButton, KanbanQuickAddIssueButton } from "./button";
import { QuickAddIssueForm } from "./form/root";

export type TQuickAddIssueButton = { isEpic?: boolean; onClick: () => void };
type Props = {
  isQuickAddOpen?: boolean;
  layout: EIssueLayoutTypes;
  prePopulatedData?: Partial<TIssue>;
  QuickAddButton?: FC<TQuickAddIssueButton>;
  customQuickAddButton?: React.ReactNode;
  containerClassName?: string;
  setIsQuickAddOpen?: (open: boolean) => void;
  quickAddCallback?: IssueQuickAddCallback;
  isEpic?: boolean;
};
export function QuickAddIssueRoot({
  isQuickAddOpen,
  layout,
  prePopulatedData,
  QuickAddButton,
  customQuickAddButton,
  containerClassName,
  setIsQuickAddOpen,
  quickAddCallback,
  isEpic = false,
}: Props) {
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { t } = useTranslation();
  const [localOpen, setLocalOpen] = useState(false);
  const open = isQuickAddOpen ?? localOpen;
  const changeOpen = setIsQuickAddOpen ?? setLocalOpen;
  if (!workspaceSlug || !projectId) return null;
  return (
    <div className={containerClassName}>
      {open ? (
        <QuickAddIssueForm
          key={`${workspaceSlug}/${projectId}`}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          layout={layout}
          prePopulatedData={prePopulatedData}
          onCreate={quickAddCallback}
          isEpic={isEpic}
          onClose={() => changeOpen(false)}
        />
      ) : (
        <>
          {QuickAddButton ? (
            <QuickAddButton isEpic={isEpic} onClick={() => changeOpen(true)} />
          ) : (
            (customQuickAddButton ?? (
              <button
                type="button"
                className="flex w-full items-center gap-2 bg-layer-transparent px-2 py-3 hover:bg-layer-transparent-hover"
                onClick={() => changeOpen(true)}
              >
                <PlusIcon className="size-3.5" />
                <span className="text-13 font-medium">{t(isEpic ? "epic.new" : "issue.new")}</span>
              </button>
            ))
          )}
        </>
      )}
    </div>
  );
}

export function GroupQuickAdd({
  layout,
  enabled,
  creationDisabled,
  groupBy,
  subGroupBy,
  prePopulate,
  quickAddCallback,
  isEpic,
}: {
  layout: EIssueLayoutTypes.LIST | EIssueLayoutTypes.KANBAN;
  enabled: boolean | undefined;
  creationDisabled: boolean;
  groupBy: TIssueGroupByOptions | undefined;
  subGroupBy?: TIssueGroupByOptions;
  prePopulate: () => Partial<TIssue>;
  quickAddCallback?: IssueQuickAddCallback;
  isEpic: boolean;
}) {
  if (creationDisabled || !canQuickAddToGroup(enabled, groupBy, subGroupBy)) return null;
  const isList = layout === EIssueLayoutTypes.LIST;
  return (
    <div
      className={isList ? "sticky bottom-0 z-[1] w-full flex-shrink-0" : "sticky bottom-0 w-full bg-surface-2 py-0.5"}
    >
      <QuickAddIssueRoot
        layout={layout}
        QuickAddButton={isList ? ListQuickAddIssueButton : KanbanQuickAddIssueButton}
        prePopulatedData={prePopulate()}
        containerClassName={isList ? "border-b border-t border-subtle bg-surface-1 " : undefined}
        quickAddCallback={quickAddCallback}
        isEpic={isEpic}
      />
    </div>
  );
}
