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
import type { TIssue, EIssueLayoutTypes } from "@plane/types";
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
  quickAddCallback?: (projectId: string | null | undefined, data: TIssue) => Promise<TIssue | undefined>;
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
