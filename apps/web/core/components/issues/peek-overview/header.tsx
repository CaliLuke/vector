/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";
import { useRef } from "react";

import { MoveDiagonal, MoveRight } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { CenterPanelIcon, CopyLinkIcon, FullScreenPanelIcon, SidePanelIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TNameDescriptionLoader } from "@plane/types";
import { CustomSelect } from "@plane/ui";
import { useWorkItemLinkCopy } from "@/hooks/use-work-item-link-copy";
import { generateWorkItemLink } from "@plane/utils/work-item";
// hooks
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useCurrentUser } from "@/hooks/use-current-user";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { IssueSubscription } from "../issue-detail/subscription";
import { WorkItemDetailQuickActions } from "../issue-layouts/quick-action-dropdowns";
import { NameDescriptionUpdateStatus } from "../issue-update-status";
import { IconButton } from "@plane/propel/icon-button";

export type TPeekModes = "side-peek" | "modal" | "full-screen";

const PEEK_OPTIONS: { key: TPeekModes; icon: typeof SidePanelIcon; i18n_title: string }[] = [
  {
    key: "side-peek",
    icon: SidePanelIcon,
    i18n_title: "common.side_peek",
  },
  {
    key: "modal",
    icon: CenterPanelIcon,
    i18n_title: "common.modal",
  },
  {
    key: "full-screen",
    icon: FullScreenPanelIcon,
    i18n_title: "common.full_screen",
  },
];

function getVisiblePeekMode(
  currentMode: (typeof PEEK_OPTIONS)[number] | undefined,
  embedIssue: boolean
): (typeof PEEK_OPTIONS)[number] | undefined {
  if (embedIssue) return undefined;
  return currentMode;
}

export type PeekOverviewHeaderProps = {
  peekMode: TPeekModes;
  setPeekMode: (value: TPeekModes) => void;
  removeRoutePeekId: () => void;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isArchived: boolean;
  disabled: boolean;
  embedIssue: boolean;
  toggleDeleteIssueModal: (value: boolean) => void;
  toggleArchiveIssueModal: (value: boolean) => void;
  toggleDuplicateIssueModal: (value: boolean) => void;
  toggleEditIssueModal: (value: boolean) => void;
  handleRestoreIssue: () => Promise<void>;
  isSubmitting: TNameDescriptionLoader;
  removeIssue: () => Promise<void>;
  archiveIssue: () => Promise<void>;
};

export const IssuePeekOverviewHeader = function IssuePeekOverviewHeader(props: PeekOverviewHeaderProps) {
  const {
    peekMode,
    setPeekMode,
    workspaceSlug,
    projectId,
    issueId,
    isArchived,
    disabled,
    embedIssue = false,
    removeRoutePeekId,
    toggleDeleteIssueModal,
    toggleArchiveIssueModal,
    toggleDuplicateIssueModal,
    toggleEditIssueModal,
    handleRestoreIssue,
    isSubmitting,
    removeIssue,
    archiveIssue,
  } = props;
  // ref
  const parentRef = useRef<HTMLDivElement>(null);
  const { t } = useTranslation();
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const record = useWorkItemDetailRecord();
  const { isMobile } = usePlatformOS();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  // derived values
  const issueDetails = issueId === record?.issueId ? record.issue : undefined;
  const currentMode = PEEK_OPTIONS.find((m) => m.key === peekMode);
  const visiblePeekMode = getVisiblePeekMode(currentMode, embedIssue);
  const projectIdentifier = projectRows?.find((project) => project.id === issueDetails?.project_id)?.identifier ?? "";
  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issueDetails?.project_id,
    issueId,
    projectIdentifier,
    sequenceId: issueDetails?.sequence_id,
    isArchived,
  });

  const copyLink = useWorkItemLinkCopy({ workspaceSlug, projectId, issueId }, workItemLink);
  const handleCopyText = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    copyLink.copy();
  };

  return (
    <div
      className={`relative flex items-center justify-between p-4 ${
        currentMode?.key === "full-screen" ? "border-b border-subtle" : ""
      }`}
    >
      <div className="flex items-center gap-4">
        <Tooltip tooltipContent={t("common.close_peek_view")} isMobile={isMobile}>
          <button onClick={removeRoutePeekId}>
            <MoveRight className="h-4 w-4 text-tertiary hover:text-secondary" />
          </button>
        </Tooltip>

        <Tooltip tooltipContent={t("issue.open_in_full_screen")} isMobile={isMobile}>
          <Link to={workItemLink} onClick={() => removeRoutePeekId()}>
            <MoveDiagonal className="h-4 w-4 text-tertiary hover:text-secondary" />
          </Link>
        </Tooltip>
        {visiblePeekMode && (
          <div className="flex flex-shrink-0 items-center gap-2">
            <CustomSelect
              value={visiblePeekMode}
              onChange={(val: TPeekModes) => setPeekMode(val)}
              customButton={
                <Tooltip tooltipContent={t("common.toggle_peek_view_layout")} isMobile={isMobile}>
                  <span className="">
                    <visiblePeekMode.icon className="h-4 w-4 text-tertiary hover:text-secondary" />
                  </span>
                </Tooltip>
              }
            >
              {PEEK_OPTIONS.map((mode) => (
                <CustomSelect.Option key={mode.key} value={mode.key}>
                  <div
                    className={`flex items-center gap-1.5 ${
                      visiblePeekMode.key === mode.key ? "text-secondary" : "text-placeholder hover:text-secondary"
                    }`}
                  >
                    <mode.icon className="-my-1 h-4 w-4 flex-shrink-0" />
                    {t(mode.i18n_title)}
                  </div>
                </CustomSelect.Option>
              ))}
            </CustomSelect>
          </div>
        )}
      </div>
      <div className="flex items-center gap-x-4">
        <NameDescriptionUpdateStatus isSubmitting={isSubmitting} />
        <div className="flex items-center gap-2">
          {currentUser && !isArchived && (
            <IssueSubscription workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} />
          )}
          <Tooltip tooltipContent={t("common.actions.copy_link")} isMobile={isMobile}>
            <IconButton
              variant="secondary"
              size="lg"
              onClick={handleCopyText}
              disabled={copyLink.disabled}
              aria-label={t("common.actions.copy_link")}
              icon={CopyLinkIcon}
            />
          </Tooltip>
          {issueDetails && (
            <WorkItemDetailQuickActions
              parentRef={parentRef}
              issue={issueDetails}
              handleDelete={removeIssue}
              handleArchive={archiveIssue}
              handleRestore={handleRestoreIssue}
              readOnly={disabled}
              toggleDeleteIssueModal={toggleDeleteIssueModal}
              toggleArchiveIssueModal={toggleArchiveIssueModal}
              toggleDuplicateIssueModal={toggleDuplicateIssueModal}
              toggleEditIssueModal={toggleEditIssueModal}
            />
          )}
        </div>
      </div>
    </div>
  );
};
