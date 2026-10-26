/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";
// plane imports
import { useTranslation } from "@plane/i18n";
import type { TIssueServiceType } from "@plane/types";
import { CollapsibleButton } from "@plane/ui";
// hooks
import { useIssueLinks } from "@/hooks/use-issue-resources";
type Props = {
  isOpen: boolean;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueServiceType: TIssueServiceType;
};

export function IssueLinksCollapsibleTitle(props: Props) {
  const { isOpen, workspaceSlug, projectId, issueId, issueServiceType } = props;
  // translation
  const { t } = useTranslation();
  const { links, query } = useIssueLinks({ workspaceSlug, projectId, issueId, serviceType: issueServiceType });
  const linksCount = query.isSuccess ? (links?.length ?? 0) : 0;

  // indicator element
  const indicatorElement = useMemo(
    () => (
      <span className="flex items-center justify-center">
        <p className="text-14 !leading-3 text-tertiary">{linksCount}</p>
      </span>
    ),
    [linksCount]
  );

  return <CollapsibleButton isOpen={isOpen} title={t("common.links")} indicatorElement={indicatorElement} />;
}
