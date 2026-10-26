/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

import { DueDatePropertyIcon } from "@plane/propel/icons";
// types
import type { TIssue } from "@plane/types";
import { cn } from "@plane/utils/common";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { shouldHighlightIssueDueDate } from "@plane/utils/work-item";
// components
import { DateDropdown } from "@/components/dropdowns/date";
// helpers
// hooks
import { useRouteProjectStates } from "@/hooks/use-project-states";

type Props = {
  issue: TIssue;
  onClose: () => void;
  onChange: (issue: TIssue, data: Partial<TIssue>, updates: any) => void;
  disabled: boolean;
};

export const SpreadsheetDueDateColumn = function SpreadsheetDueDateColumn(props: Props) {
  const { issue, onChange, disabled, onClose } = props;
  // store hooks
  const { getStateById } = useRouteProjectStates(issue.project_id);
  // derived values
  const stateDetails = getStateById(issue.state_id);

  return (
    <div className="h-11 border-b-[0.5px] border-subtle">
      <DateDropdown
        value={issue.target_date}
        minDate={getDate(issue.start_date)}
        onChange={(data) => {
          const targetDate = data ? renderFormattedPayloadDate(data) : null;
          onChange(
            issue,
            { target_date: targetDate },
            {
              changed_property: "target_date",
              change_details: targetDate,
            }
          );
        }}
        disabled={disabled}
        placeholder="Due date"
        icon={<DueDatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
        buttonVariant="transparent-with-text"
        buttonContainerClassName="w-full"
        buttonClassName={cn(
          "rounded-none px-page-x text-left group-[.selected-issue-row]:bg-accent-primary/5 group-[.selected-issue-row]:hover:bg-accent-primary/10",
          {
            "text-danger-primary": shouldHighlightIssueDueDate(issue.target_date, stateDetails?.group),
          }
        )}
        optionsClassName="z-[9]"
        clearIconClassName="!text-primary"
        onClose={onClose}
      />
    </div>
  );
};
