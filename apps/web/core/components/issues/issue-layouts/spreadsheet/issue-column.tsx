/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";

// types
import type { TIssueStandardDisplayProperty, IIssueDisplayProperties, TIssue } from "@plane/types";
// constants
import { SPREADSHEET_COLUMNS } from "../utils";
// components
import { shouldRenderColumn } from "@/helpers/issue-filter.helper";
import { WithDisplayPropertiesHOC } from "../properties/with-display-properties-HOC";

type Props = {
  displayProperties: IIssueDisplayProperties;
  issueDetail: TIssue;
  disableUserActions: boolean;
  property: TIssueStandardDisplayProperty;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  isEstimateEnabled: boolean;
};

export const IssueColumn = function IssueColumn(props: Props) {
  const { displayProperties, issueDetail, disableUserActions, property, updateIssue, isEstimateEnabled } = props;
  // router
  const tableCellRef = useRef<HTMLTableCellElement | null>(null);

  const shouldRenderProperty = shouldRenderColumn(property, isEstimateEnabled);

  const Column = SPREADSHEET_COLUMNS[property];

  if (!Column) return null;

  const handleUpdateIssue = async (issue: TIssue, data: Partial<TIssue>) => {
    if (updateIssue) await updateIssue(issue.project_id, issue.id, data);
  };

  return (
    <WithDisplayPropertiesHOC
      displayProperties={displayProperties}
      displayPropertyKey={property}
      shouldRenderProperty={() => shouldRenderProperty}
    >
      <td
        tabIndex={0}
        className="h-11 min-w-36 border-r-[1px] border-subtle text-13 after:absolute after:bottom-[-1px] after:w-full after:border after:border-subtle"
        ref={tableCellRef}
      >
        <Column
          issue={issueDetail}
          onChange={handleUpdateIssue}
          disabled={disableUserActions}
          onClose={() => tableCellRef?.current?.focus()}
        />
      </td>
    </WithDisplayPropertiesHOC>
  );
};
