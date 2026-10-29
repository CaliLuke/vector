/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { workItemServiceType } from "@/helpers/work-item-service-type";
import { useRef, useState } from "react";

import { createPortal } from "react-dom";
// plane imports
import type { EditorRefApi } from "@plane/editor";
import { EIssueServiceType } from "@plane/types";
import { cn } from "@plane/utils/common";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import useKeypress from "@/hooks/use-keypress";
import usePeekOverviewOutsideClickDetector from "@/hooks/use-peek-overview-outside-click";
// local imports
import type { TIssuePeekOperations } from "./root";
import { useTextSaveStatus } from "../use-text-save-status";
import { IssueActivity } from "../issue-detail/issue-activity";
import { IssueDetailWidgets } from "../issue-detail-widgets";
import { IssuePeekOverviewError } from "./error";
import type { TPeekModes } from "./header";
import { IssuePeekOverviewHeader } from "./header";
import { PeekOverviewIssueDetails } from "./issue-detail";
import { IssuePeekOverviewLoader } from "./loader";
import { PeekOverviewProperties } from "./properties";

interface IIssueView {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isLoading?: boolean;
  isError?: boolean;
  is_archived: boolean;
  disabled?: boolean;
  embedIssue?: boolean;
  embedRemoveCurrentNotification?: () => void;
  issueOperations: TIssuePeekOperations;
}

type TLocalPeekModal = "delete" | "archive" | "duplicate" | "edit";

export const IssueView = function IssueView(props: IIssueView) {
  const {
    workspaceSlug,
    projectId,
    issueId,
    isLoading,
    isError,
    is_archived,
    disabled = false,
    embedIssue = false,
    embedRemoveCurrentNotification,
    issueOperations,
  } = props;
  // states
  const [peekMode, setPeekMode] = useState<TPeekModes>("side-peek");
  const {
    status: isSubmitting,
    setTitleStatus,
    setDescriptionStatus,
  } = useTextSaveStatus(`${workspaceSlug}/${projectId}/${issueId}`);
  const [openModal, setOpenModal] = useState<TLocalPeekModal | undefined>();
  // ref
  const issuePeekOverviewRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<EditorRefApi>(null);
  // store hooks
  const { setPeekIssue, isAnyModalOpen } = useIssueDetailUiState();
  const { isAnyModalOpen: isAnyEpicModalOpen } = useIssueDetailUiState(EIssueServiceType.EPICS);
  const record = useWorkItemDetailRecord();
  const issue = issueId === record?.issueId ? record.issue : undefined;
  // remove peek id
  const removeRoutePeekId = () => {
    setPeekIssue(undefined);
    if (embedIssue && embedRemoveCurrentNotification) embedRemoveCurrentNotification();
  };

  const toggleDeleteIssueModal = createModalToggle(setOpenModal, "delete");
  const toggleArchiveIssueModal = createModalToggle(setOpenModal, "archive");
  const toggleDuplicateIssueModal = createModalToggle(setOpenModal, "duplicate");
  const toggleEditIssueModal = createModalToggle(setOpenModal, "edit");

  const isAnyLocalModalOpen = openModal !== undefined;

  usePeekOverviewOutsideClickDetector(
    issuePeekOverviewRef,
    () => {
      const isAnyDropbarOpen = editorRef.current?.isAnyDropbarOpen();
      if (!embedIssue) {
        if (!isAnyModalOpen && !isAnyEpicModalOpen && !isAnyLocalModalOpen && !isAnyDropbarOpen) {
          removeRoutePeekId();
        }
      }
    },
    issueId,
    ["main-sidebar"]
  );

  const handleKeyDown = () => {
    const editorImageFullScreenModalElement = document.querySelector(".editor-image-full-screen-modal");
    const dropdownElement = document.activeElement?.tagName === "INPUT";
    const isAnyDropbarOpen = editorRef.current?.isAnyDropbarOpen();
    if (
      !isAnyModalOpen &&
      !isAnyEpicModalOpen &&
      !isAnyLocalModalOpen &&
      !dropdownElement &&
      !isAnyDropbarOpen &&
      !editorImageFullScreenModalElement
    ) {
      removeRoutePeekId();
      const issueElement = document.getElementById(`issue-${issueId}`);
      if (issueElement) issueElement?.focus();
    }
  };

  useKeypress("Escape", () => !embedIssue && handleKeyDown());

  const handleRestore = async () => {
    await issueOperations.restore(workspaceSlug, projectId, issueId);
  };

  const peekOverviewIssueClassName = cn(
    !embedIssue
      ? "absolute z-[25] flex flex-col overflow-hidden rounded-sm border border-subtle bg-surface-1 transition-all duration-300"
      : `h-full w-full`,
    !embedIssue && {
      "top-0 right-0 bottom-0 w-full border-0 border-l md:w-[50%]": peekMode === "side-peek",
      "top-[8.33%] left-[8.33%] size-5/6": peekMode === "modal",
      "absolute inset-0 m-4": peekMode === "full-screen",
    }
  );

  const shouldUsePortal = !embedIssue;

  const portalContainer = document.getElementById("full-screen-portal") as HTMLElement;

  const itemDetails = (
    <PeekOverviewIssueDetails
      editorRef={editorRef}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      issueId={issueId}
      issueOperations={issueOperations}
      disabled={disabled}
      isArchived={is_archived}
      onEditorOpenChange={toggleEditIssueModal}
      setTitleStatus={setTitleStatus}
      setDescriptionStatus={setDescriptionStatus}
    />
  );

  const content = (
    <div className="w-full text-body-sm-regular">
      {issueId && (
        <div
          ref={issuePeekOverviewRef}
          className={peekOverviewIssueClassName}
          style={{
            boxShadow:
              "0px 4px 8px 0px rgba(0, 0, 0, 0.12), 0px 6px 12px 0px rgba(16, 24, 40, 0.12), 0px 1px 16px 0px rgba(16, 24, 40, 0.12)",
          }}
        >
          {isError ? (
            <div className="relative h-screen w-full overflow-hidden">
              <IssuePeekOverviewError removeRoutePeekId={removeRoutePeekId} />
            </div>
          ) : (
            isLoading && <IssuePeekOverviewLoader removeRoutePeekId={removeRoutePeekId} />
          )}
          {!isLoading && !isError && issue && (
            <>
              {/* header */}
              <IssuePeekOverviewHeader
                peekMode={peekMode}
                setPeekMode={(value) => setPeekMode(value)}
                removeRoutePeekId={removeRoutePeekId}
                toggleDeleteIssueModal={toggleDeleteIssueModal}
                toggleArchiveIssueModal={toggleArchiveIssueModal}
                toggleDuplicateIssueModal={toggleDuplicateIssueModal}
                handleRestoreIssue={handleRestore}
                isArchived={is_archived}
                issueId={issueId}
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                isSubmitting={isSubmitting}
                removeIssue={() => issueOperations.remove(workspaceSlug, projectId, issueId)}
                archiveIssue={() => issueOperations.archive(workspaceSlug, projectId, issueId)}
                disabled={disabled}
                embedIssue={embedIssue}
              />
              {/* content */}
              <div className="vertical-scrollbar relative scrollbar-md h-full w-full overflow-hidden overflow-y-auto">
                {["side-peek", "modal"].includes(peekMode) ? (
                  <div className="relative flex flex-col gap-3 space-y-3 px-8 py-5">
                    {itemDetails}

                    <div className="py-2">
                      <IssueDetailWidgets
                        workspaceSlug={workspaceSlug}
                        projectId={projectId}
                        issueId={issueId}
                        disabled={disabled || is_archived}
                        issueServiceType={EIssueServiceType.ISSUES}
                      />
                    </div>

                    <PeekOverviewProperties
                      workspaceSlug={workspaceSlug}
                      projectId={projectId}
                      issueId={issueId}
                      issueOperations={issueOperations}
                      disabled={disabled || is_archived}
                    />

                    <IssueActivity
                      serviceType={workItemServiceType(issue)}
                      workspaceSlug={workspaceSlug}
                      projectId={projectId}
                      issueId={issueId}
                      disabled={is_archived}
                    />
                  </div>
                ) : (
                  <div className="vertical-scrollbar flex h-full w-full overflow-auto">
                    <div className="relative h-full w-full space-y-6 overflow-auto p-4 py-5">
                      <div className="space-y-3">
                        {itemDetails}

                        <div className="py-2">
                          <IssueDetailWidgets
                            workspaceSlug={workspaceSlug}
                            projectId={projectId}
                            issueId={issueId}
                            disabled={disabled || is_archived}
                            issueServiceType={EIssueServiceType.ISSUES}
                          />
                        </div>

                        <IssueActivity
                          serviceType={workItemServiceType(issue)}
                          workspaceSlug={workspaceSlug}
                          projectId={projectId}
                          issueId={issueId}
                          disabled={is_archived}
                        />
                      </div>
                    </div>
                    <div className="vertical-scrollbar scrollbar-sm h-full !w-[400px] flex-shrink-0 overflow-hidden border-l border-subtle p-4 py-5">
                      <PeekOverviewProperties
                        workspaceSlug={workspaceSlug}
                        projectId={projectId}
                        issueId={issueId}
                        issueOperations={issueOperations}
                        disabled={disabled || is_archived}
                      />
                    </div>
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );

  return <>{shouldUsePortal && portalContainer ? createPortal(content, portalContainer) : content}</>;
};

function createModalToggle(
  setOpenModal: React.Dispatch<React.SetStateAction<TLocalPeekModal | undefined>>,
  modal: TLocalPeekModal
) {
  return (isOpen: boolean) =>
    setOpenModal((current) => {
      if (isOpen) return modal;
      return current === modal ? undefined : current;
    });
}
