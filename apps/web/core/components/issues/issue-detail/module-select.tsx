/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { cn } from "@plane/utils/common";
import { ModuleDropdown } from "@/components/dropdowns/module/dropdown";
import { useIssueAction } from "@/hooks/use-issue-action";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import type { TIssueOperations } from "./root";

type TIssueModuleSelect = {
  className?: string;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  disabled?: boolean;
};

export const IssueModuleSelect = function IssueModuleSelect(props: TIssueModuleSelect) {
  const { className = "", workspaceSlug, projectId, issueId, issueOperations, disabled = false } = props;
  const { t } = useTranslation();
  const { issue } = useWorkItemDetail(workspaceSlug, projectId, issueId);
  const updateModules = issueOperations.changeModulesInIssue;
  const change = useIssueAction(
    { workspaceSlug, projectId, issueId, context: "detail" },
    !disabled && !!issue && !!updateModules
  );
  const handleIssueModuleChange = (moduleIds: string[]) => {
    if (!issue || !updateModules) return;
    const selected = new Set(moduleIds);
    const current = new Set(issue.module_ids ?? []);
    const added = [...selected].filter((id) => !current.has(id));
    const removed = [...current].filter((id) => !selected.has(id));
    if (added.length === 0 && removed.length === 0) return;
    change.run(() => updateModules(workspaceSlug, projectId, issueId, added, removed));
  };

  return (
    <div className={cn(`flex h-full items-center gap-1`, className)}>
      <ModuleDropdown
        projectId={projectId}
        value={issue?.module_ids ?? []}
        onChange={handleIssueModuleChange}
        placeholder={t("module.no_module")}
        disabled={change.disabled}
        className="group h-full w-full"
        buttonContainerClassName="w-full text-left rounded-sm"
        buttonClassName={`text-body-xs-medium justify-between ${issue?.module_ids?.length ? "" : "text-placeholder"}`}
        buttonVariant="transparent-with-text"
        hideIcon
        dropdownArrow
        dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
        multiple
        itemClassName="px-2"
      />
    </div>
  );
};
