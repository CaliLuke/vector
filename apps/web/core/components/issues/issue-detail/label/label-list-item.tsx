/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import { CloseIcon, LabelFilledIcon } from "@plane/propel/icons";
// types
import { useProjectLabels } from "@/hooks/use-labels";
import type { TLabelOperations } from "./root";

type TLabelListItem = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  labelId: string;
  values: string[];
  labelOperations: TLabelOperations;
  disabled: boolean;
};

export function LabelListItem(props: TLabelListItem) {
  const { workspaceSlug, projectId, issueId, labelId, values, labelOperations, disabled } = props;
  // hooks
  const { getLabelById } = useProjectLabels(workspaceSlug, projectId);

  const label = getLabelById(labelId);

  const removal = useMutation({
    mutationFn: async () => {
      if (values && !disabled) {
        const currentLabels = values.filter((_labelId) => _labelId !== labelId);
        await labelOperations.updateIssue(workspaceSlug, projectId, issueId, { label_ids: currentLabels });
      }
    },
  });

  if (!label) return <></>;
  return (
    <Button
      variant="tertiary"
      size="sm"
      key={labelId}
      onClick={() => removal.mutate()}
      disabled={disabled || removal.isPending}
    >
      <LabelFilledIcon className="size-3" color={label.color ?? "#000000"} />
      <span className="text-body-xs-regular">{label.name}</span>
      {!disabled && <CloseIcon className="h-2.5 w-2.5 transition-all group-hover:text-danger-primary" />}
    </Button>
  );
}
