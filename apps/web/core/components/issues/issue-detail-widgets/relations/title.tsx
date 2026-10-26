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
import { useIssueRelations } from "@/hooks/use-issue-resources";
// components
import { useTimeLineRelationOptions } from "@/components/relations";
import { countVisibleRelations } from "./count";
type Props = {
  isOpen: boolean;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueServiceType?: TIssueServiceType;
};

export function RelationsCollapsibleTitle(props: Props) {
  const { isOpen, workspaceSlug, projectId, issueId, issueServiceType = EIssueServiceType.ISSUES } = props;
  const { t } = useTranslation();
  const { relations } = useIssueRelations({ workspaceSlug, projectId, issueId, serviceType: issueServiceType });
  const ISSUE_RELATION_OPTIONS = useTimeLineRelationOptions();
  // derived values
  const relationsCount = countVisibleRelations(relations, ISSUE_RELATION_OPTIONS);

  // indicator element
  const indicatorElement = useMemo(
    () => (
      <span className="flex items-center justify-center">
        <p className="text-14 !leading-3 text-tertiary">{relationsCount}</p>
      </span>
    ),
    [relationsCount]
  );

  return <CollapsibleButton isOpen={isOpen} title={t("common.relations")} indicatorElement={indicatorElement} />;
}
