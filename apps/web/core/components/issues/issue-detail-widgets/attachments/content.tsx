/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import type { TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// local imports
import { IssueAttachmentItemList } from "../../attachment/attachment-item-list";
import type { TAttachmentHelpers } from "./helper";

type Props = {
  workspaceSlug: string;
  disabled: boolean;
  issueServiceType?: TIssueServiceType;
  attachmentHelpers: TAttachmentHelpers;
};

export function IssueAttachmentsCollapsibleContent(props: Props) {
  const { workspaceSlug, disabled, issueServiceType = EIssueServiceType.ISSUES, attachmentHelpers } = props;
  return (
    <IssueAttachmentItemList
      workspaceSlug={workspaceSlug}
      disabled={disabled}
      attachmentHelpers={attachmentHelpers}
      issueServiceType={issueServiceType}
    />
  );
}
