/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import type { TWorkspaceDraftIssue } from "@plane/types";
import { AlertModalCore } from "@plane/ui";
import { useWorkspacePermissions } from "@/hooks/use-permissions";

import { isIssueCreator } from "../issue-delete-modal-feedback";

import { useIssueConfirmation, useIssueConfirmationScope } from "../use-issue-confirmation";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  data?: TWorkspaceDraftIssue;
  onSubmit?: () => Promise<void>;
};

export function WorkspaceDraftIssueDeleteIssueModal(props: Props) {
  const { data, isOpen } = props;
  const { workspaceSlug, scopeKey } = useIssueConfirmationScope([data?.id, data?.created_by]);

  if (!isOpen || !data?.id || !workspaceSlug) return null;
  return (
    <WorkspaceDraftIssueDeleteDialog key={scopeKey} {...props} workspaceSlug={workspaceSlug} scopeKey={scopeKey} />
  );
}

type DialogProps = Props & { workspaceSlug: string; scopeKey: string };

function WorkspaceDraftIssueDeleteDialog(props: DialogProps) {
  const { data: issue, handleClose, isOpen, onSubmit, workspaceSlug, scopeKey } = props;
  const permissions = useWorkspacePermissions(workspaceSlug);
  const { t } = useTranslation();
  const confirmation = useIssueConfirmation(scopeKey, permissions, handleClose);

  if (!issue) return null;

  const isCreator = isIssueCreator(confirmation.currentUser?.id, issue.created_by);
  const authorized =
    confirmation.canStart &&
    isCreator &&
    permissions.can([EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST]);

  const retryAccess = () => {
    void Promise.all([confirmation.currentUserQuery.refetch(), permissions.membershipQuery.refetch()]);
  };
  const handleIssueDelete = () => {
    confirmation.submitDelete({
      authorized,
      onSubmit,
      entity: "work item",
      successToast: {
        title: `${t("success")}!`,
        message: t("workspace_draft_issues.toasts.deleted.success"),
      },
      t,
    });
  };

  return (
    <AlertModalCore
      handleClose={confirmation.closeWhenIdle}
      handleSubmit={handleIssueDelete}
      isSubmitting={confirmation.isPending || !confirmation.canStart || !onSubmit}
      isOpen={isOpen}
      title={t("workspace_draft_issues.delete_modal.title")}
      content={
        confirmation.accessError ? (
          <span>
            Unable to verify delete permissions. Please retry or close this dialog.{" "}
            <button type="button" onClick={retryAccess} disabled={confirmation.isPending}>
              Retry
            </button>
          </span>
        ) : (
          t("workspace_draft_issues.delete_modal.description")
        )
      }
      primaryButtonText={{
        loading: confirmation.canStart ? t("deleting") : "Checking permissions",
        default: t("delete"),
      }}
      secondaryButtonText={t("cancel")}
    />
  );
}
