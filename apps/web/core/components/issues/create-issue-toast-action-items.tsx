/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { useClipboardCopy } from "@plane/hooks";

import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { copyUrlToClipboard } from "@plane/utils/string";
import { generateWorkItemLink } from "@plane/utils/work-item";
// plane imports
// helpers
// hooks
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { EIssueServiceType } from "@plane/types";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

type TCreateIssueToastActionItems = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isEpic?: boolean;
};

export const CreateIssueToastActionItems = function CreateIssueToastActionItems(props: TCreateIssueToastActionItems) {
  const { workspaceSlug, issueId, isEpic = false } = props;
  const { copied, copy } = useClipboardCopy(JSON.stringify([workspaceSlug, props.projectId, issueId, isEpic]), 3000);
  // store hooks
  const issueData = useWorkItemDetail(
    workspaceSlug,
    props.projectId,
    issueId,
    isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;

  // derived values
  const issue = issueData.issue;
  const projectIdentifier = projectRows?.find((project) => project.id === issue?.project_id)?.identifier ?? "";

  if (!issue) return null;

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issue?.project_id,
    issueId,
    projectIdentifier,
    sequenceId: issue?.sequence_id,
    isEpic,
  });

  const copyToClipboard = async (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    e.preventDefault();
    e.stopPropagation();
    if ((await copy(() => copyUrlToClipboard(workItemLink))) === "failed") {
      setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy link", message: "Please try again." });
    }
  };

  return (
    <div className="-ml-2 flex items-center gap-1 text-11 text-secondary">
      <a
        href={workItemLink}
        target="_blank"
        rel="noopener noreferrer"
        className="rounded-sm px-2 py-1 font-medium text-accent-primary hover:bg-surface-2"
      >
        {`View ${isEpic ? "epic" : "work item"}`}
      </a>

      {copied ? (
        <>
          <span className="cursor-default px-2 py-1 text-secondary">Copied!</span>
        </>
      ) : (
        <>
          <button
            className="hidden cursor-pointer rounded-sm px-2 py-1 text-tertiary group-hover:flex hover:bg-surface-2 hover:text-secondary"
            onClick={copyToClipboard}
          >
            Copy link
          </button>
        </>
      )}
    </div>
  );
};

export function notifyIssueCreated({
  title,
  message,
  ...issue
}: TCreateIssueToastActionItems & { title: string; message: string }) {
  setToast({ type: TOAST_TYPE.SUCCESS, title, message, actionItems: <CreateIssueToastActionItems {...issue} /> });
}
