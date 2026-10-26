/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
// hooks
import { useProjectLabels } from "@/hooks/use-labels";
import { useIntakePreferences } from "@/providers/intake-preferences";
import { InboxAppliedFilterGroup, toggleAppliedFilterValue } from "./shared";

function LabelIcons({ color }: { color: string }) {
  return <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: color }} />;
}

export const InboxIssueAppliedFiltersLabel = function InboxIssueAppliedFiltersLabel() {
  // hooks
  const { filters, setFilter } = useIntakePreferences();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { getLabelById } = useProjectLabels(workspaceSlug?.toString(), projectId?.toString());
  // derived values
  const filteredValues = filters?.labels || [];
  const currentOptionDetail = (labelId: string) => getLabelById(labelId) || undefined;

  const clearFilter = () => setFilter("labels", undefined);
  const options = filteredValues.flatMap((value) => {
    const optionDetail = currentOptionDetail(value);
    if (!optionDetail) return [];
    return [
      {
        key: value,
        leading: (
          <div className="relative flex h-3 w-3 flex-shrink-0 items-center justify-center overflow-hidden">
            <LabelIcons color={optionDetail.color} />
          </div>
        ),
        label: optionDetail.name,
        onRemove: () => setFilter("labels", toggleAppliedFilterValue(filteredValues, value)),
      },
    ];
  });

  return (
    <InboxAppliedFilterGroup
      label="Label"
      options={options}
      hasFilters={filteredValues.length > 0}
      onClearAll={clearFilter}
    />
  );
};
