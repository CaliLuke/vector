/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIssueAction } from "@/hooks/use-issue-action";
import { useTranslation } from "@plane/i18n";
import { CycleIcon, EstimatePropertyIcon, ModuleIcon } from "@plane/propel/icons";
import type { TIssue } from "@plane/types";
import { EstimateDropdown } from "@/components/dropdowns/estimate";
import { SidebarPropertyListItem } from "@/components/common/layout/sidebar/property-list-item";
import type { TIssueOperations } from "./root";
import { IssueCycleSelect } from "./cycle-select";
import { IssueModuleSelect } from "./module-select";

type IssueProjectPropertiesProps = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  issue: TIssue;
  disabled: boolean;
  estimateEnabled: boolean;
  estimateButtonClassName: string;
  estimatePlaceholder: string;
  moduleView: boolean | undefined;
  cycleView: boolean | undefined;
};

export const IssueProjectProperties = function IssueProjectProperties(props: IssueProjectPropertiesProps) {
  const {
    workspaceSlug,
    projectId,
    issueId,
    issueOperations,
    issue,
    disabled,
    estimateEnabled,
    estimateButtonClassName,
    estimatePlaceholder,
    moduleView,
    cycleView,
  } = props;
  const { t } = useTranslation();
  const propertyChange = useIssueAction({ workspaceSlug, projectId, issueId, context: "detail" }, !disabled);

  return (
    <>
      {estimateEnabled && (
        <SidebarPropertyListItem icon={EstimatePropertyIcon} label={t("common.estimate")}>
          <EstimateDropdown
            value={issue.estimate_point ?? undefined}
            onChange={(value: string | undefined) =>
              propertyChange.run(() =>
                issueOperations.update(workspaceSlug, projectId, issueId, { estimate_point: value })
              )
            }
            projectId={projectId}
            disabled={propertyChange.disabled}
            buttonVariant="transparent-with-text"
            className="group w-full grow"
            buttonContainerClassName="w-full text-left h-7.5"
            buttonClassName={estimateButtonClassName}
            placeholder={estimatePlaceholder}
            hideIcon
            dropdownArrow
            dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
          />
        </SidebarPropertyListItem>
      )}
      {moduleView && (
        <SidebarPropertyListItem icon={ModuleIcon} label={t("common.modules")}>
          <IssueModuleSelect
            className="w-full grow"
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            issueId={issueId}
            issueOperations={issueOperations}
            disabled={disabled}
          />
        </SidebarPropertyListItem>
      )}
      {cycleView && (
        <SidebarPropertyListItem icon={CycleIcon} label={t("common.cycle")} appendElement={null}>
          <IssueCycleSelect
            className="h-7.5 w-full grow"
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            issueId={issueId}
            issueOperations={issueOperations}
            disabled={disabled}
          />
        </SidebarPropertyListItem>
      )}
    </>
  );
};
