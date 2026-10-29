/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIssueAction } from "@/hooks/use-issue-action";
import { useRouter } from "@tanstack/react-router";

import { MinusCircle } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import type { TBaseIssue, TIssue } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// component
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// ui
import { ControlLink, CustomMenu } from "@plane/ui";
// helpers
import { generateWorkItemLink } from "@plane/utils/work-item";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkspaceStates } from "@/hooks/use-project-states";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useWorkItemPeek } from "@/providers/issue-detail-ui-state";
// types
import type { TIssueOperations } from "../root";
import { IssueParentSiblings } from "./siblings";

export type TIssueParentDetail = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issue: TIssue;
  issueOperations: TIssueOperations;
  disabled: boolean;
};

type TParentIssueSummary = Partial<TBaseIssue> & { id: string; project_id: string; sequence_id: number };

function hasParentSummaryIdentity(issue: Partial<TBaseIssue> | undefined): issue is TParentIssueSummary {
  return Boolean(issue?.id && issue.project_id && issue.sequence_id !== undefined);
}

export function IssueParentDetail(props: TIssueParentDetail) {
  const { workspaceSlug, projectId, issue, issueOperations, disabled } = props;
  const parentId = issue.parent?.id ?? issue.parent_id ?? undefined;
  const parentProjectId = issue.parent?.project_id ?? projectId;
  const parentServiceType = issue.parent?.is_epic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES;
  const parentRead = useWorkItemDetail(
    workspaceSlug,
    parentProjectId,
    parentId,
    parentServiceType,
    Boolean(parentId && !hasParentSummaryIdentity(issue.parent))
  );

  // derived values
  const parentIssue = hasParentSummaryIdentity(issue.parent) ? issue.parent : parentRead.issue;

  if (!hasParentSummaryIdentity(parentIssue)) return null;

  return (
    <IssueParentSummary
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      currentIssue={issue}
      parentIssue={parentIssue}
      issueOperations={issueOperations}
      disabled={disabled}
    />
  );
}

function IssueParentSummary({
  workspaceSlug,
  projectId,
  currentIssue,
  parentIssue,
  issueOperations,
  disabled,
}: {
  workspaceSlug: string;
  projectId: string;
  currentIssue: TIssue;
  parentIssue: TParentIssueSummary;
  issueOperations: TIssueOperations;
  disabled: boolean;
}) {
  const propertyChange = useIssueAction(
    { workspaceSlug, projectId, issueId: currentIssue.id, context: "detail" },
    !disabled
  );
  const router = useRouter();
  const { t } = useTranslation();
  const { getProjectStates } = useWorkspaceStates(workspaceSlug);
  const { peekIssue, setPeekIssue } = useWorkItemPeek();
  const { isMobile } = usePlatformOS();
  const { projects } = useWorkspaceProjectDetails();
  const isParentEpic = parentIssue.is_epic;
  const projectIdentifier = projects?.find((project) => project.id === parentIssue.project_id)?.identifier ?? "";
  const issueParentState = getProjectStates(parentIssue.project_id)?.find(
    (state) => state?.id === parentIssue.state_id
  );
  const stateColor = issueParentState?.color || undefined;

  const workItemLink = projectIdentifier
    ? generateWorkItemLink({
        workspaceSlug,
        projectId: parentIssue.project_id,
        issueId: parentIssue.id,
        projectIdentifier,
        sequenceId: parentIssue.sequence_id,
        isEpic: isParentEpic,
      })
    : undefined;

  const handleParentIssueClick = () => {
    if (!workItemLink) return;
    if (isParentEpic) void router.navigate({ href: workItemLink });
    else if (!peekIssue || peekIssue.issueId !== parentIssue.id) {
      if (isMobile) void router.navigate({ href: workItemLink });
      else setPeekIssue({ workspaceSlug, projectId: parentIssue.project_id, issueId: parentIssue.id });
    }
  };

  return (
    <>
      <div className="mb-5 flex w-min items-center gap-3 rounded-md border border-strong bg-layer-1 px-2.5 py-1 text-11 whitespace-nowrap">
        {workItemLink ? (
          <ControlLink href={workItemLink} onClick={handleParentIssueClick}>
            <ParentIdentity
              parentIssue={parentIssue}
              projectId={parentIssue.project_id}
              projectIdentifier={projectIdentifier}
              stateColor={stateColor}
            />
          </ControlLink>
        ) : (
          <ParentIdentity
            parentIssue={parentIssue}
            projectId={parentIssue.project_id}
            projectIdentifier={projectIdentifier}
            stateColor={stateColor}
          />
        )}

        <CustomMenu ellipsis optionsClassName="p-1.5">
          <div className="border-b border-strong text-11 font-medium text-secondary">{t("issue.sibling.label")}</div>

          <IssueParentSiblings
            workspaceSlug={workspaceSlug}
            currentIssueId={currentIssue.id}
            parentIssueId={parentIssue.id}
            parentProjectId={parentIssue.project_id}
          />

          <CustomMenu.MenuItem
            disabled={propertyChange.disabled}
            onClick={() => {
              propertyChange.run(() =>
                issueOperations.update(workspaceSlug, projectId, currentIssue.id, { parent_id: null })
              );
            }}
            className="flex items-center gap-2 py-2 text-danger-primary"
          >
            <MinusCircle className="h-4 w-4" />
            <span>{t("issue.remove.parent.label")}</span>
          </CustomMenu.MenuItem>
        </CustomMenu>
      </div>
    </>
  );
}

function ParentIdentity({
  parentIssue,
  projectId,
  projectIdentifier,
  stateColor,
}: {
  parentIssue: TParentIssueSummary;
  projectId: string;
  projectIdentifier: string;
  stateColor: string | undefined;
}) {
  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-2.5">
        <span className="block h-2 w-2 rounded-md" style={{ backgroundColor: stateColor }} />
        {projectIdentifier && (
          <IssueIdentifier
            projectId={projectId}
            issueTypeId={parentIssue.type_id}
            projectIdentifier={projectIdentifier}
            issueSequenceId={parentIssue.sequence_id}
            size="xs"
            variant="secondary"
          />
        )}
      </div>
      <span className="truncate text-primary">{(parentIssue.name ?? "").substring(0, 50)}</span>
    </div>
  );
}
