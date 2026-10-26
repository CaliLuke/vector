/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation } from "@tanstack/react-query";
// components
import type { TLabelOperations } from "../root";
import { IssueLabelSelect } from "./label-select";
// types

type TIssueLabelSelectRoot = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  values: string[];
  labelOperations: TLabelOperations;
};

export function IssueLabelSelectRoot(props: TIssueLabelSelectRoot) {
  const { workspaceSlug, projectId, issueId, values, labelOperations } = props;

  const selection = useMutation({
    mutationFn: async (_labelIds: string[]) => {
      await labelOperations.updateIssue(workspaceSlug, projectId, issueId, { label_ids: _labelIds });
    },
  });

  return (
    <IssueLabelSelect
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      issueId={issueId}
      values={values}
      onSelect={selection.mutate}
      onAddLabel={labelOperations.createLabel}
    />
  );
}
