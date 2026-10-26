/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";

import { useTranslation } from "@plane/i18n";
import type { TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { CollapsibleButton } from "@plane/ui";
// hooks
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
type Props = {
  isOpen: boolean;
  issueId: string;
  issueServiceType?: TIssueServiceType;
};

export const IssueAttachmentsCollapsibleTitle = function IssueAttachmentsCollapsibleTitle(props: Props) {
  const { isOpen, issueId, issueServiceType = EIssueServiceType.ISSUES } = props;
  const { t } = useTranslation();
  // store hooks
  // derived values
  const record = useWorkItemDetailRecord();
  const issue = record?.issueId === issueId && record.serviceType === issueServiceType ? record.issue : undefined;
  const attachmentCount = issue?.attachment_count ?? 0;

  // indicator element
  const indicatorElement = useMemo(
    () => (
      <span className="flex items-center justify-center">
        <p className="text-14 !leading-3 text-tertiary">{attachmentCount}</p>
      </span>
    ),
    [attachmentCount]
  );

  return <CollapsibleButton isOpen={isOpen} title={t("common.attachments")} indicatorElement={indicatorElement} />;
};
