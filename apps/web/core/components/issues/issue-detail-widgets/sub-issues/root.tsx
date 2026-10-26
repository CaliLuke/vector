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
import { SubIssuesCollapsibleContent } from "./content";
import { SubIssuesCollapsibleTitle } from "./title";
import { SubWorkItemTitleActions } from "./title-actions";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled?: boolean;
  issueServiceType: TIssueServiceType;
};

export const SubIssuesCollapsible = function SubIssuesCollapsible(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled = false, issueServiceType } = props;
  // store hooks
  const { openWidgets, toggleOpenWidget } = useIssueDetailUiState(issueServiceType);
  // derived values
  const isCollapsibleOpen = openWidgets.includes("sub-work-items");

  return (
    <Collapsible
      isOpen={isCollapsibleOpen}
      onToggle={() => toggleOpenWidget("sub-work-items")}
      title={
        <SubIssuesCollapsibleTitle
          isOpen={isCollapsibleOpen}
          parentIssueId={issueId}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueServiceType={issueServiceType}
        />
      }
      titleActionElement={
        <SubWorkItemTitleActions
          projectId={projectId}
          parentId={issueId}
          disabled={disabled}
          issueServiceType={issueServiceType}
        />
      }
      buttonClassName="w-full"
    >
      <SubIssuesCollapsibleContent
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        parentIssueId={issueId}
        disabled={disabled}
        issueServiceType={issueServiceType}
      />
    </Collapsible>
  );
};
