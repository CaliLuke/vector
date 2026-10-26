/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import type { TDeDupeIssue, TIssue } from "@plane/types";
import { AlertModalCore } from "@plane/ui";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

import { canDeleteIssue, isIssueCreator } from "./issue-delete-modal-feedback";

import { useIssueConfirmation, useIssueConfirmationScope } from "./use-issue-confirmation";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  data?: TIssue | TDeDupeIssue;
  isSubIssue?: boolean;
  onSubmit?: () => Promise<void>;
  isEpic?: boolean;
};

export function DeleteIssueModal(props: Props) {
  const { data, isOpen, isSubIssue = false, isEpic = false } = props;
  const { workspaceSlug, scopeKey } = useIssueConfirmationScope([data?.project_id, data?.id, isSubIssue, isEpic]);

  if (!isOpen || !data?.id || !workspaceSlug) return null;
  return <DeleteIssueDialog key={scopeKey} {...props} workspaceSlug={workspaceSlug} scopeKey={scopeKey} />;
}

type DialogProps = Props & { workspaceSlug: string; scopeKey: string };

function DeleteIssueDialog(props: DialogProps) {
  const { data, isOpen, handleClose, isSubIssue = false, onSubmit, isEpic = false, workspaceSlug, scopeKey } = props;
  const issue = data;
  const permissions = useProjectPermissions(workspaceSlug, issue?.project_id ?? undefined);
  const projectCatalogue = useWorkspaceProjectDetails();
  const { t } = useTranslation();
  const confirmation = useIssueConfirmation(scopeKey, permissions, handleClose);

  if (!issue) return null;
  const projectDetails = projectCatalogue.projects?.find((project) => project.id === issue.project_id);
  const accountId = confirmation.currentUser?.id;

  const isCreator = isIssueCreator(accountId, issue.created_by);
  const authorized = canDeleteIssue({
    canStart: confirmation.canStart,
    hasProjectRole: permissions.projectRole !== undefined,
    isCreator,
    isAdmin: permissions.can([EUserPermissions.ADMIN]),
  });
  const issueLabels = getIssueDeleteLabels(t, isEpic, isSubIssue);

  const handleIssueDelete = () => {
    confirmation.submitDelete({
      authorized,
      onSubmit,
      entity: issueLabels.entity,
      successToast: {
        title: t("common.success"),
        message: t("entity.delete.success", { entity: issueLabels.translatedEntity }),
      },
      t,
    });
  };
  const retryAccess = () => {
    void Promise.all([
      confirmation.currentUserQuery.refetch(),
      permissions.membershipQuery.refetch(),
      permissions.rolesQuery.refetch(),
    ]);
  };

  return (
    <AlertModalCore
      handleClose={confirmation.closeWhenIdle}
      handleSubmit={handleIssueDelete}
      isSubmitting={confirmation.isPending || !confirmation.canStart || !onSubmit}
      isOpen={isOpen}
      title={t("entity.delete.label", { entity: issueLabels.translatedIssueType })}
      content={
        <>
          {confirmation.accessError ? (
            <span>
              Unable to verify delete permissions. Please retry or close this dialog.{" "}
              <button type="button" onClick={retryAccess} disabled={confirmation.isPending}>
                Retry
              </button>
            </span>
          ) : (
            <>
              {/* TODO: Translate here */}
              {`Are you sure you want to delete ${issueLabels.issueType} `}
              <span className="font-medium break-words text-primary">
                {projectDetails?.identifier}-{issue.sequence_id}
              </span>
              {` ? All of the data related to the ${issueLabels.issueType} will be permanently removed. This action cannot be undone.`}
            </>
          )}
        </>
      }
      primaryButtonText={{ loading: confirmation.canStart ? "Deleting" : "Checking permissions", default: "Delete" }}
    />
  );
}

function getIssueDeleteLabels(t: (key: string) => string, isEpic: boolean, isSubIssue: boolean) {
  const issueType = isEpic ? "epic" : "work item";
  const translatedIssueType = isEpic ? t("common.epic") : t("common.work_item");
  return {
    issueType,
    translatedIssueType,
    translatedEntity: isSubIssue ? t("common.sub_work_item") : translatedIssueType,
    entity: isSubIssue ? "sub-work item" : issueType,
  };
}
