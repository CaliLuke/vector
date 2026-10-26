/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
import type { TIssue, TIssueServiceType, TIssueRelationTypes } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { Collapsible } from "@plane/ui";
// components
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import { useTimeLineRelationOptions } from "@/components/relations";
import { CreateUpdateEpicModal } from "@/components/epic-modal";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useIssueRelations } from "@/hooks/use-issue-resources";
// helper
import { DeleteIssueModal } from "../../delete-issue-modal";
import { RelationIssueList } from "../../relations/issue-list";
import { useRelationOperations } from "./helper";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
  issueServiceType?: TIssueServiceType;
};

type TIssueCrudState = { toggle: boolean; issueId: string | undefined; issue: TIssue | undefined };

type TVisibleRelationRow = {
  relationKey: TIssueRelationTypes;
  issues: TIssue[];
  icon: ((size: number) => React.ReactElement) | undefined;
  labelKey: string | undefined;
  className: string | undefined;
};

function getVisibleRelationRows(
  relations: ReturnType<typeof useIssueRelations>["relations"],
  options: ReturnType<typeof useTimeLineRelationOptions>
): TVisibleRelationRow[] {
  if (!relations) return [];
  return (Object.keys(relations) as TIssueRelationTypes[]).flatMap((relationKey) => {
    const option = options[relationKey];
    const issues = relations[relationKey];
    if (!option || issues.length === 0) return [];
    return [{ relationKey, issues, icon: option.icon, labelKey: option.i18n_label, className: option.className }];
  });
}

export type TRelationObject = {
  key: TIssueRelationTypes;
  i18n_label: string;
  className: string;
  icon: (size: number) => React.ReactElement;
  placeholder: string;
};

function RelationRetryButton({
  queryError,
  accessError,
  retry,
  refetch,
}: {
  queryError: boolean;
  accessError: boolean;
  retry: () => Promise<unknown>;
  refetch: () => Promise<unknown>;
}) {
  const { t } = useTranslation();
  if (!queryError && !accessError) return null;
  return (
    <button
      type="button"
      className="text-11 text-danger-primary"
      onClick={() => void (accessError ? retry() : refetch())}
    >
      {t("common.retry")}
    </button>
  );
}

export const RelationsCollapsibleContent = function RelationsCollapsibleContent(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled = false, issueServiceType = EIssueServiceType.ISSUES } = props;
  // plane hooks
  const { t } = useTranslation();
  // state
  const [issueCrudState, setIssueCrudState] = useState<{
    update: TIssueCrudState;
    delete: TIssueCrudState;
    removeRelation: TIssueCrudState & { relationKey: string | undefined; relationIssueId: string | undefined };
  }>({
    update: {
      toggle: false,
      issueId: undefined,
      issue: undefined,
    },
    delete: {
      toggle: false,
      issueId: undefined,
      issue: undefined,
    },
    removeRelation: {
      toggle: false,
      issueId: undefined,
      issue: undefined,
      relationKey: undefined,
      relationIssueId: undefined,
    },
  });

  // store hooks
  const { toggleDeleteIssueModal, toggleCreateIssueModal } = useIssueDetailUiState(issueServiceType);

  // helper
  const issueOperations = useRelationOperations(workspaceSlug, projectId, issueId);
  const epicOperations = useRelationOperations(workspaceSlug, projectId, issueId, EIssueServiceType.EPICS);

  // derived values
  const { relations, query, isError, retry } = useIssueRelations({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: issueServiceType,
  });
  const ISSUE_RELATION_OPTIONS = useTimeLineRelationOptions();

  const handleIssueCrudState = (
    key: "update" | "delete" | "removeRelation",
    _issueId: string | null,
    issue: TIssue | null = null,
    relationKey?: TIssueRelationTypes | null,
    relationIssueId?: string | null
  ) => {
    setIssueCrudState((prevState) => ({
      ...prevState,
      [key]: {
        toggle: !prevState[key].toggle,
        issueId: _issueId,
        issue: issue,
        relationKey: relationKey,
        relationIssueId: relationIssueId,
      },
    }));
  };

  // if relations are not available, return null
  const visibleRelations = getVisibleRelationRows(relations, ISSUE_RELATION_OPTIONS);

  const shouldRenderIssueDeleteModal =
    issueCrudState?.delete?.toggle &&
    issueCrudState?.delete?.issue &&
    issueCrudState.delete.issueId &&
    issueCrudState.delete.issue.id;

  const shouldRenderIssueUpdateModal = issueCrudState?.update?.toggle && issueCrudState?.update?.issue;

  return (
    <>
      <RelationRetryButton queryError={query.isError} accessError={isError} retry={retry} refetch={query.refetch} />
      <div className="gap- flex flex-col">
        {visibleRelations.map((relation) => (
          <div key={relation.relationKey}>
            <Collapsible
              buttonClassName="w-full"
              title={
                <div className={`flex h-9 w-full items-center gap-1 px-2.5 py-1 ${relation.className}`}>
                  <span>{relation.icon ? relation.icon(14) : null}</span>
                  <span className="text-13 leading-5 font-medium">{relation.labelKey ? t(relation.labelKey) : ""}</span>
                </div>
              }
              defaultOpen
            >
              <RelationIssueList
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                issueId={issueId}
                relationKey={relation.relationKey}
                issues={relation.issues}
                disabled={disabled}
                handleIssueCrudState={handleIssueCrudState}
                issueServiceType={issueServiceType}
              />
            </Collapsible>
          </div>
        ))}
      </div>

      {shouldRenderIssueDeleteModal && (
        <DeleteIssueModal
          isOpen={issueCrudState?.delete?.toggle}
          handleClose={() => {
            handleIssueCrudState("delete", null, null);
            toggleDeleteIssueModal(null);
          }}
          data={issueCrudState?.delete?.issue as TIssue}
          onSubmit={async () => {
            if (
              issueCrudState.delete.issue &&
              issueCrudState.delete.issue.id &&
              issueCrudState.delete.issue.project_id
            ) {
              const deleteOperation = issueCrudState.delete.issue?.is_epic
                ? epicOperations.remove
                : issueOperations.remove;
              await deleteOperation(
                workspaceSlug,
                issueCrudState.delete.issue?.project_id,
                issueCrudState?.delete?.issue?.id
              );
              void query.refetch().catch(() => undefined);
            }
          }}
          isEpic={!!issueCrudState.delete.issue?.is_epic}
        />
      )}

      {shouldRenderIssueUpdateModal && (
        <>
          {issueCrudState?.update?.issue?.is_epic ? (
            <CreateUpdateEpicModal
              isOpen={issueCrudState?.update?.toggle}
              onClose={() => {
                handleIssueCrudState("update", null, null);
                toggleCreateIssueModal(false);
              }}
              data={issueCrudState?.update?.issue ?? undefined}
              onSubmit={async (_issue: TIssue) => {
                if (!_issue.id || !_issue.project_id) return;
                await epicOperations.update(workspaceSlug, _issue.project_id, _issue.id, _issue);
              }}
            />
          ) : (
            <CreateUpdateIssueModal
              isOpen={issueCrudState?.update?.toggle}
              onClose={() => {
                handleIssueCrudState("update", null, null);
                toggleCreateIssueModal(false);
              }}
              data={issueCrudState?.update?.issue ?? undefined}
              onSubmit={async (_issue: TIssue) => {
                if (!_issue.id || !_issue.project_id) return;
                await issueOperations.update(workspaceSlug, _issue.project_id, _issue.id, _issue);
              }}
            />
          )}
        </>
      )}
    </>
  );
};
