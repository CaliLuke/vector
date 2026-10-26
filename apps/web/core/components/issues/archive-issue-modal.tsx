/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useParams } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";

import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TDeDupeIssue, TIssue } from "@plane/types";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

import { submitIssueArchiveConfirmation } from "./issue-delete-modal-feedback";

import { useIssueConfirmation } from "./use-issue-confirmation";

type Props = {
  data?: TIssue | TDeDupeIssue;
  handleClose: () => void;
  isOpen: boolean;
  onSubmit?: () => Promise<void>;
};

export function ArchiveIssueModal(props: Props) {
  const { data, isOpen } = props;
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const pathname = useLocation({ select: (location) => location.pathname });
  const client = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const generation = useIdentityGeneration(client);

  if (!isOpen || !data?.id || !workspaceSlug) return null;
  const scopeKey = JSON.stringify([workspaceSlug, pathname, data.project_id, data.id, currentUser?.id, generation]);
  return <ArchiveIssueDialog key={scopeKey} {...props} workspaceSlug={workspaceSlug} scopeKey={scopeKey} />;
}

type DialogProps = Props & { workspaceSlug: string; scopeKey: string };

function ArchiveIssueDialog(props: DialogProps) {
  const { data, handleClose, isOpen, onSubmit, workspaceSlug, scopeKey } = props;
  const issue = data;
  const permissions = useProjectPermissions(workspaceSlug, issue?.project_id ?? undefined);
  const projectCatalogue = useWorkspaceProjectDetails();
  const { t } = useTranslation();
  const {
    currentUserQuery,
    isCurrent,
    canStart,
    accessError,
    isPending: isArchiving,
    pending,
    setPending,
    closeWhenIdle,
  } = useIssueConfirmation(scopeKey, permissions, handleClose);

  if (!issue) return null;
  const projectDetails = projectCatalogue.projects?.find((project) => project.id === issue.project_id);

  const canArchive = Boolean(
    canStart &&
    permissions.projectRole !== undefined &&
    permissions.can([EUserPermissions.ADMIN, EUserPermissions.MEMBER]) &&
    onSubmit
  );

  const retryAccess = () => {
    void Promise.all([
      currentUserQuery.refetch(),
      permissions.membershipQuery.refetch(),
      permissions.rolesQuery.refetch(),
    ]);
  };

  const archiveIssue = () => {
    void submitIssueArchiveConfirmation({
      canArchive,
      isPending: () => pending.current,
      setPending,
      isCurrent,
      onSubmit,
      onFailure: () => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("common.error.label"),
          message: t("issue.archive.failed.message"),
        });
      },
      onSuccess: () => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("issue.archive.success.label"),
          message: t("issue.archive.success.message"),
        });
        handleClose();
      },
    });
  };

  return (
    <ArchiveIssueDialogContent
      isOpen={isOpen}
      issue={issue}
      projectIdentifier={projectDetails?.identifier}
      isArchiving={isArchiving}
      canStart={canStart}
      canArchive={canArchive}
      accessError={accessError}
      onRetry={retryAccess}
      onClose={closeWhenIdle}
      onArchive={archiveIssue}
      t={t}
    />
  );
}

function ArchiveIssueDialogContent({
  isOpen,
  issue,
  projectIdentifier,
  isArchiving,
  canStart,
  canArchive,
  accessError,
  onRetry,
  onClose,
  onArchive,
  t,
}: {
  isOpen: boolean;
  issue: TIssue | TDeDupeIssue;
  projectIdentifier?: string;
  isArchiving: boolean;
  canStart: boolean;
  canArchive: boolean;
  accessError: boolean;
  onRetry: () => void;
  onClose: () => void;
  onArchive: () => void;
  t: (key: string) => string;
}) {
  return (
    <ModalCore isOpen={isOpen} handleClose={onClose} position={EModalPosition.CENTER} width={EModalWidth.LG}>
      <div className="px-5 py-4">
        <h3 className="text-18 font-medium 2xl:text-20">
          {t("issue.archive.label")} {projectIdentifier} {issue.sequence_id}
        </h3>
        <p className="mt-3 text-13 text-secondary">{t("issue.archive.confirm_message")}</p>
        {accessError && (
          <div role="alert" className="mt-3 text-13 text-danger-primary">
            Unable to verify archive permissions. Please retry.
            <button type="button" className="ml-2 underline" onClick={onRetry} disabled={isArchiving}>
              Retry
            </button>
          </div>
        )}
        {!canStart && !accessError && <p className="mt-3 text-13 text-secondary">Checking archive permissions…</p>}
        {canStart && !canArchive && (
          <p className="mt-3 text-13 text-danger-primary">You do not have permission to archive this work item.</p>
        )}
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={onClose} disabled={isArchiving}>
            {t("common.cancel")}
          </Button>
          <Button
            variant="primary"
            size="lg"
            onClick={onArchive}
            loading={isArchiving}
            disabled={!canArchive || isArchiving}
          >
            {isArchiving ? t("common.archiving") : t("common.archive")}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
