/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import type { TSpreadsheetColumnProps } from "@plane/types";
// components
import { EstimateDropdown } from "@/components/dropdowns/estimate";

export const SpreadsheetEstimateColumn = function SpreadsheetEstimateColumn(props: TSpreadsheetColumnProps) {
  const { issue, onChange, disabled, onClose } = props;

  return (
    <div className="h-11 border-b-[0.5px] border-subtle">
      <EstimateDropdown
        value={issue.estimate_point || undefined}
        onChange={(data) =>
          onChange(issue, { estimate_point: data }, { changed_property: "estimate_point", change_details: data })
        }
        placeholder="Estimate"
        projectId={issue.project_id ?? undefined}
        disabled={disabled}
        buttonVariant="transparent-with-text"
        buttonClassName="text-left rounded-none group-[.selected-issue-row]:bg-accent-primary/5 group-[.selected-issue-row]:hover:bg-accent-primary/10 px-page-x"
        buttonContainerClassName="w-full"
        onClose={onClose}
      />
    </div>
  );
};
