/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

// plane imports
import type { TIssueServiceType } from "@plane/types";
import { Collapsible } from "@plane/ui";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
// local imports
import { IssueAttachmentsCollapsibleContent } from "./content";
import { IssueAttachmentsCollapsibleTitle } from "./title";
import { IssueAttachmentActionButton } from "./quick-action-button";
import type { TAttachmentHelpers } from "./helper";

type Props = {
  workspaceSlug: string;
  issueId: string;
  disabled?: boolean;
  issueServiceType: TIssueServiceType;
  attachmentHelpers: TAttachmentHelpers;
};

export const AttachmentsCollapsible = function AttachmentsCollapsible(props: Props) {
  const { workspaceSlug, issueId, disabled = false, issueServiceType, attachmentHelpers } = props;
  // store hooks
  const { openWidgets, toggleOpenWidget } = useIssueDetailUiState(issueServiceType);

  // derived values
  const isCollapsibleOpen = openWidgets.includes("attachments");

  return (
    <Collapsible
      isOpen={isCollapsibleOpen}
      onToggle={() => toggleOpenWidget("attachments")}
      title={
        <IssueAttachmentsCollapsibleTitle
          isOpen={isCollapsibleOpen}
          issueId={issueId}
          issueServiceType={issueServiceType}
        />
      }
      titleActionElement={
        !disabled && (
          <IssueAttachmentActionButton
            workspaceSlug={workspaceSlug}
            disabled={disabled}
            attachmentHelpers={attachmentHelpers}
          />
        )
      }
      buttonClassName="w-full"
    >
      <IssueAttachmentsCollapsibleContent
        workspaceSlug={workspaceSlug}
        disabled={disabled}
        issueServiceType={issueServiceType}
        attachmentHelpers={attachmentHelpers}
      />
    </Collapsible>
  );
};
