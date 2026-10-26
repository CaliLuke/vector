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
import { RelationsCollapsibleContent } from "./content";
import { RelationsCollapsibleTitle } from "./title";
import { RelationActionButton } from "./quick-action-button";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled?: boolean;
  issueServiceType: TIssueServiceType;
};

export const RelationsCollapsible = function RelationsCollapsible(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled = false, issueServiceType } = props;
  // store hooks
  const { openWidgets, toggleOpenWidget } = useIssueDetailUiState(issueServiceType);
  // derived values
  const isCollapsibleOpen = openWidgets.includes("relations");

  return (
    <Collapsible
      isOpen={isCollapsibleOpen}
      onToggle={() => toggleOpenWidget("relations")}
      title={
        <RelationsCollapsibleTitle
          isOpen={isCollapsibleOpen}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          issueServiceType={issueServiceType}
        />
      }
      titleActionElement={
        !disabled && <RelationActionButton issueId={issueId} disabled={disabled} issueServiceType={issueServiceType} />
      }
      buttonClassName="w-full"
    >
      <RelationsCollapsibleContent
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        disabled={disabled}
        issueServiceType={issueServiceType}
      />
    </Collapsible>
  );
};
