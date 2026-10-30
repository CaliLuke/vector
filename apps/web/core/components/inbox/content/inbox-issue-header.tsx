/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { useRouter } from "@tanstack/react-router";
import { useCallback, useEffect, useRef, useState } from "react";
import { MoveRight } from "lucide-react";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { IconButton } from "@plane/propel/icon-button";
import { ChevronDownIcon, ChevronUpIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TNameDescriptionLoader } from "@plane/types";
import { EInboxIssueStatus } from "@plane/types";
import { Row } from "@plane/ui/row";
import { copyUrlToClipboard } from "@plane/utils/string";
import { findHowManyDaysLeft } from "@plane/utils/datetime";
import { generateWorkItemLink } from "@plane/utils/work-item";
// components
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import { NameDescriptionUpdateStatus } from "@/components/issues/issue-update-status";
// hooks
import { useProjectDetail } from "@/hooks/use-projects";
import { useIntakeMutation } from "@/hooks/use-intake";
import { EInboxIssueCurrentTab } from "@plane/types";
import type { TInboxIssueCurrentTab } from "@plane/types";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";

// store
import type { TInboxIssue } from "@plane/types";
// local imports
import { InboxIssueStatus } from "../inbox-issue-status";
import { DeclineIssueModal } from "../modals/decline-issue-modal";
import { DeleteInboxIssueModal } from "../modals/delete-issue-modal";
import { WorkItemSelectionModal } from "@/components/issues/work-item-selection-modal";
import { InboxIssueSnoozeModal } from "../modals/snooze-issue-modal";
import { InboxIssueActionsMobileHeader } from "./inbox-issue-mobile-header";

import { IntakeDesktopActions } from "./intake-action-controls";
import type { IntakeActions } from "./intake-action-controls";

const handleActionWithPermission = (isAdmin: boolean, action: () => void, errorMessage: string) => {
  if (isAdmin) action();
  else {
    setToast({
      type: TOAST_TYPE.ERROR,
      title: "Permission denied",
      message: errorMessage,
    });
  }
};

const EMPTY_ISSUE_IDS: string[] = [];

function useIntakeActionPermissions(workspaceSlug: string, projectId: string, inboxIssue: TInboxIssue) {
  const { data: currentUser } = useCurrentUser();
  const { can } = useProjectPermissions(workspaceSlug, projectId);
  const isProjectAdmin = can([EUserPermissions.ADMIN]);
  const isAllowed = can([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  return {
    isProjectAdmin,
    isAllowed,
    canTriage:
      isAllowed && (inboxIssue.status === EInboxIssueStatus.PENDING || inboxIssue.status === EInboxIssueStatus.SNOOZED),
    canDelete: isProjectAdmin || Boolean(currentUser?.id && inboxIssue.issue.created_by === currentUser.id),
  };
}

type TInboxIssueActionsHeader = {
  workspaceSlug: string;
  projectId: string;
  inboxIssue: TInboxIssue;
  navigation?: { tab: TInboxIssueCurrentTab; issueIds: string[] };
  isSubmitting: TNameDescriptionLoader;
  isMobileSidebar: boolean;
  setIsMobileSidebar: (value: boolean) => void;
  isNotificationEmbed: boolean;
  embedRemoveCurrentNotification?: () => void;
  onDeleteAccepted: () => void;
};

export function InboxIssueActionsHeader(props: TInboxIssueActionsHeader) {
  const client = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const generation = useIdentityGeneration(client);
  const scopeKey = JSON.stringify([
    currentUser?.id,
    generation,
    props.workspaceSlug,
    props.projectId,
    props.inboxIssue.id,
    props.inboxIssue.issue.id,
    props.navigation?.tab,
    props.isNotificationEmbed,
  ]);
  const mounted = useCommittedScope(scopeKey);
  const identity = captureIdentityScope(client, currentUser?.id);
  const isCurrent = () => mounted() && isIdentityScopeCurrent(client, identity);
  return <InboxIssueActions key={scopeKey} {...props} isCurrent={isCurrent} />;
}

function InboxIssueActions(props: TInboxIssueActionsHeader & { isCurrent: () => boolean }) {
  const { isCurrent } = props;
  const {
    workspaceSlug,
    projectId,
    inboxIssue,
    isSubmitting,
    isMobileSidebar,
    setIsMobileSidebar,
    isNotificationEmbed,
    embedRemoveCurrentNotification,
  } = props;
  // states
  const [isSnoozeDateModalOpen, setIsSnoozeDateModalOpen] = useState(false);
  const [selectDuplicateIssue, setSelectDuplicateIssue] = useState(false);
  const [acceptIssueModal, setAcceptIssueModal] = useState(false);
  const acceptNavigation = useRef<{ nextId: string | undefined } | undefined>(undefined);
  const [declineIssueModal, setDeclineIssueModal] = useState(false);
  const [deleteIssueModal, setDeleteIssueModal] = useState(false);
  // store
  const { mutateAsync } = useIntakeMutation(workspaceSlug, projectId);
  const currentTab = props.navigation?.tab ?? EInboxIssueCurrentTab.OPEN;
  const filteredInboxIssueIds = props.navigation?.issueIds ?? EMPTY_ISSUE_IDS;
  const updateStatus = (data: Partial<Pick<TInboxIssue, "status" | "snoozed_till" | "duplicate_to">>) =>
    mutateAsync({ kind: "status", issueId: inboxIssue.issue.id, data });
  const { isAllowed, isProjectAdmin, canTriage, canDelete } = useIntakeActionPermissions(
    workspaceSlug,
    projectId,
    inboxIssue
  );
  const { data: currentProjectDetails } = useProjectDetail(workspaceSlug, projectId);
  const { t } = useTranslation();
  const projectIdentifier = currentProjectDetails?.identifier;

  const router = useRouter();

  const issue = inboxIssue.issue;
  // derived values
  const isAcceptedOrDeclined = [-1, 1, 2].includes(inboxIssue.status);
  // days left for snooze
  const numberOfDaysLeft = findHowManyDaysLeft(inboxIssue.snoozed_till);
  const isSnoozed = Boolean(inboxIssue.snoozed_till && numberOfDaysLeft && numberOfDaysLeft > 0);

  const currentInboxIssueId = inboxIssue.issue.id;

  const redirectIssue = (): string | undefined => {
    let nextOrPreviousIssueId: string | undefined = undefined;
    const currentIssueIndex = filteredInboxIssueIds.findIndex((id) => id === currentInboxIssueId);
    if (filteredInboxIssueIds[currentIssueIndex + 1])
      nextOrPreviousIssueId = filteredInboxIssueIds[currentIssueIndex + 1];
    else if (filteredInboxIssueIds[currentIssueIndex - 1])
      nextOrPreviousIssueId = filteredInboxIssueIds[currentIssueIndex - 1];
    else nextOrPreviousIssueId = undefined;
    return nextOrPreviousIssueId;
  };

  const handleRedirection = (nextOrPreviousIssueId: string | undefined) => {
    if (!isCurrent()) return;
    if (!isNotificationEmbed) {
      if (nextOrPreviousIssueId)
        void router.navigate({
          href: `/${workspaceSlug}/projects/${projectId}/intake?currentTab=${currentTab}&inboxIssueId=${nextOrPreviousIssueId}`,
        });
      else void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/intake?currentTab=${currentTab}` });
    }
  };

  const handleInboxIssueAccept = async () => {
    acceptNavigation.current ??= { nextId: redirectIssue() };
    if (inboxIssue.status !== EInboxIssueStatus.ACCEPTED) await updateStatus({ status: EInboxIssueStatus.ACCEPTED });
  };

  const handleAcceptedIssueSaved = async () => {
    if (!isCurrent()) return;
    setAcceptIssueModal(false);
    handleRedirection(acceptNavigation.current?.nextId);
    acceptNavigation.current = undefined;
  };

  const handleInboxIssueDecline = async () => {
    await updateStatus({ status: EInboxIssueStatus.DECLINED });
  };

  const handleInboxIssueSnooze = async (date: Date) => {
    const nextOrPreviousIssueId = redirectIssue();
    await updateStatus({ status: EInboxIssueStatus.SNOOZED, snoozed_till: date });
    if (!isCurrent()) return;
    setIsSnoozeDateModalOpen(false);
    handleRedirection(nextOrPreviousIssueId);
  };

  const handleInboxIssueDuplicate = async (issueId: string) => {
    await updateStatus({ status: EInboxIssueStatus.DUPLICATE, duplicate_to: issueId });
  };

  const handleInboxIssueDelete = async () => {
    if (!currentInboxIssueId) return;
    await mutateAsync({ kind: "delete", issueId: currentInboxIssueId });
    props.onDeleteAccepted();
  };

  const handleIssueSnoozeAction = async () => {
    if (isSnoozed) {
      const nextOrPreviousIssueId = redirectIssue();
      try {
        await updateStatus({ status: EInboxIssueStatus.PENDING, snoozed_till: null });
        handleRedirection(nextOrPreviousIssueId);
      } catch {
        if (!isCurrent()) return;
        setToast({ type: TOAST_TYPE.ERROR, title: "Could not unsnooze work item", message: "Please try again." });
      }
    } else {
      setIsSnoozeDateModalOpen(true);
    }
  };

  const handleCopyIssueLink = async (path: string) => {
    try {
      await copyUrlToClipboard(path);
      if (!isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("common.link_copied"),
        message: t("common.copied_to_clipboard"),
      });
    } catch {
      if (!isCurrent()) return;
      setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy to clipboard", message: "Please try again." });
    }
  };

  const currentIssueIndex = filteredInboxIssueIds.findIndex((issueId) => issueId === currentInboxIssueId);

  const handleInboxIssueNavigation = useCallback(
    (direction: "next" | "prev") => {
      if (!filteredInboxIssueIds || !currentInboxIssueId) return;
      const activeElement = document.activeElement;
      if (activeElement && (activeElement.classList.contains("tiptap") || activeElement.tagName === "TEXTAREA")) return;
      const nextIssueIndex =
        direction === "next"
          ? (currentIssueIndex + 1) % filteredInboxIssueIds.length
          : (currentIssueIndex - 1 + filteredInboxIssueIds.length) % filteredInboxIssueIds.length;
      const nextIssueId = filteredInboxIssueIds[nextIssueIndex];
      if (!nextIssueId) return;
      void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/intake?inboxIssueId=${nextIssueId}` });
    },
    [currentInboxIssueId, currentIssueIndex, filteredInboxIssueIds, projectId, router, workspaceSlug]
  );

  const onKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "ArrowUp") {
        handleInboxIssueNavigation("prev");
      } else if (e.key === "ArrowDown") {
        handleInboxIssueNavigation("next");
      }
    },
    [handleInboxIssueNavigation]
  );

  useEffect(() => {
    if (isSubmitting !== "submitting" && !isNotificationEmbed) {
      document.addEventListener("keydown", onKeyDown);
      return () => document.removeEventListener("keydown", onKeyDown);
    }
    return undefined;
  }, [onKeyDown, isNotificationEmbed, isSubmitting]);

  const workItemLink = generateWorkItemLink({
    workspaceSlug: workspaceSlug,
    projectId: issue.project_id,
    issueId: currentInboxIssueId,
    projectIdentifier: projectIdentifier,
    sequenceId: issue.sequence_id,
  });

  const actions: IntakeActions = {
    canTriage,
    canDelete,
    isAllowed,
    isClosed: isAcceptedOrDeclined,
    isSnoozed,
    workItemLink,
    accept: () =>
      handleActionWithPermission(
        isProjectAdmin,
        () => setAcceptIssueModal(true),
        t("inbox_issue.errors.accept_permission")
      ),
    decline: () =>
      handleActionWithPermission(
        isProjectAdmin,
        () => setDeclineIssueModal(true),
        t("inbox_issue.errors.decline_permission")
      ),
    snooze: () =>
      handleActionWithPermission(
        isProjectAdmin,
        () => {
          void handleIssueSnoozeAction();
        },
        t("inbox_issue.errors.snooze_permission")
      ),
    duplicate: () =>
      handleActionWithPermission(
        isProjectAdmin,
        () => setSelectDuplicateIssue(true),
        "Only project admins can mark work item as duplicate"
      ),
    remove: () => setDeleteIssueModal(true),
    copy: () => handleCopyIssueLink(workItemLink),
    open: () => {
      void router.navigate({ href: workItemLink });
    },
  };

  return (
    <>
      <>
        <WorkItemSelectionModal
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={currentInboxIssueId}
          isOpen={selectDuplicateIssue}
          onClose={() => setSelectDuplicateIssue(false)}
          selectedId={inboxIssue.duplicate_to}
          onSelect={(selected) => handleInboxIssueDuplicate(selected.id)}
        />
        <CreateUpdateIssueModal
          data={inboxIssue.issue}
          isOpen={acceptIssueModal}
          onClose={() => {
            setAcceptIssueModal(false);
            acceptNavigation.current = undefined;
          }}
          beforeFormSubmit={handleInboxIssueAccept}
          onSubmit={handleAcceptedIssueSaved}
          withDraftIssueWrapper={false}
          fetchIssueDetails={false}
          showActionItemsOnUpdate
          modalTitle={t("inbox_issue.actions.move", {
            value: `${projectIdentifier}-${issue.sequence_id}`,
          })}
          primaryButtonText={{
            default: t("add_to_project"),
            loading: t("adding"),
          }}
        />
        <DeclineIssueModal
          projectIdentifier={projectIdentifier}
          data={inboxIssue.issue}
          isOpen={declineIssueModal}
          onClose={() => setDeclineIssueModal(false)}
          onSubmit={handleInboxIssueDecline}
          onSuccess={() => handleRedirection(redirectIssue())}
          isCurrent={isCurrent}
        />
        <DeleteInboxIssueModal
          projectIdentifier={projectIdentifier}
          data={inboxIssue.issue}
          isOpen={deleteIssueModal}
          onClose={() => setDeleteIssueModal(false)}
          onSubmit={handleInboxIssueDelete}
          isCurrent={isCurrent}
        />
        <InboxIssueSnoozeModal
          isOpen={isSnoozeDateModalOpen}
          handleClose={() => {
            if (isCurrent()) setIsSnoozeDateModalOpen(false);
          }}
          value={inboxIssue.snoozed_till ?? undefined}
          onConfirm={handleInboxIssueSnooze}
        />
      </>

      <Row className="relative z-15 hidden h-full w-full items-center justify-between gap-2 border-b border-subtle bg-surface-1 lg:flex">
        <div className="flex items-center gap-4">
          {isNotificationEmbed && (
            <button onClick={embedRemoveCurrentNotification}>
              <MoveRight className="h-4 w-4 text-tertiary hover:text-secondary" />
            </button>
          )}
          {issue.project_id && issue.sequence_id && (
            <h3 className="flex-shrink-0 text-14 font-medium text-tertiary">
              {projectIdentifier}-{issue.sequence_id}
            </h3>
          )}
          <InboxIssueStatus inboxIssue={inboxIssue} iconSize={12} />
          <div className="flex w-full items-center justify-end">
            <NameDescriptionUpdateStatus isSubmitting={isSubmitting} />
          </div>
        </div>

        <div className="flex items-center gap-2">
          {!isNotificationEmbed && (
            <div className="flex items-center gap-x-2">
              <IconButton
                variant="secondary"
                size="lg"
                icon={ChevronUpIcon}
                aria-label="Previous work item"
                onClick={() => handleInboxIssueNavigation("prev")}
              />
              <IconButton
                variant="secondary"
                size="lg"
                icon={ChevronDownIcon}
                aria-label="Next work item"
                onClick={() => handleInboxIssueNavigation("next")}
              />
            </div>
          )}

          <IntakeDesktopActions actions={actions} />
        </div>
      </Row>

      <div className="lg:hidden">
        <InboxIssueActionsMobileHeader
          inboxIssue={inboxIssue}
          isSubmitting={isSubmitting}
          actions={actions}
          handleInboxIssueNavigation={handleInboxIssueNavigation}
          isMobileSidebar={isMobileSidebar}
          setIsMobileSidebar={setIsMobileSidebar}
          isNotificationEmbed={isNotificationEmbed}
          embedRemoveCurrentNotification={embedRemoveCurrentNotification}
        />
      </div>
    </>
  );
}
