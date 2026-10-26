/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
// plane types
import type { IIssueLabel, TIssue } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import { PowerKLabelsMenu } from "@/components/power-k/menus/labels";
// hooks
import { useProjectLabels } from "@/hooks/use-labels";

type Props = {
  handleSelect: (label: IIssueLabel) => void;
  workItemDetails: TIssue;
};

export function PowerKWorkItemLabelsMenu(props: Props) {
  const { handleSelect, workItemDetails } = props;
  // store hooks
  const { workspaceSlug } = useParams({ strict: false });
  const { labels: projectLabels } = useProjectLabels(
    workspaceSlug?.toString(),
    workItemDetails.project_id ?? undefined
  );
  // derived values
  const filteredLabelsList = projectLabels;

  if (!filteredLabelsList) return <Spinner />;

  return <PowerKLabelsMenu labels={filteredLabelsList} onSelect={handleSelect} value={workItemDetails.label_ids} />;
}
