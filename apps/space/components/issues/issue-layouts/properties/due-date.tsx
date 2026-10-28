/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useParams } from "react-router";

import { DueDatePropertyIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
// helpers
import { renderFormattedDate } from "@/helpers/date-time.helper";
import { shouldHighlightIssueDueDate } from "@/helpers/issue.helper";
// hooks
import { statesOptions } from "@/queries/lookups";

type Props = {
  due_date: string | undefined;
  stateId: string | undefined;
  shouldHighLight?: boolean;
  shouldShowBorder?: boolean;
};

export const IssueBlockDate = function IssueBlockDate(props: Props) {
  const { due_date, stateId, shouldHighLight = true, shouldShowBorder = true } = props;
  const { data: states = [] } = useRecordCollection(statesOptions(useParams().anchor));

  const state = states.find((item) => item.id === stateId);

  const formattedDate = renderFormattedDate(due_date);

  return (
    <Tooltip tooltipHeading="Due Date" tooltipContent={formattedDate}>
      <div
        className={cn("flex h-full items-center gap-1 rounded-sm px-2.5 py-1 text-11 text-primary", {
          "text-danger-primary": shouldHighLight && due_date && shouldHighlightIssueDueDate(due_date, state?.group),
          "border-[0.5px] border-strong": shouldShowBorder,
        })}
      >
        <DueDatePropertyIcon className="size-3 flex-shrink-0" />
        {formattedDate ? formattedDate : "No Date"}
      </div>
    </Tooltip>
  );
};
