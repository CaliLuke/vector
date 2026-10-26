/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssueLayoutTypes } from "@plane/types";
import { LayoutSelection } from "./layout-selection";
import { MobileLayoutSelection } from "./mobile-layout-selection";

const PARENT_ISSUE_LAYOUTS: EIssueLayoutTypes[] = [
  EIssueLayoutTypes.LIST,
  EIssueLayoutTypes.KANBAN,
  EIssueLayoutTypes.CALENDAR,
  EIssueLayoutTypes.SPREADSHEET,
  EIssueLayoutTypes.GANTT,
];
export const PARENT_ISSUE_MOBILE_LAYOUTS: EIssueLayoutTypes[] = [
  EIssueLayoutTypes.LIST,
  EIssueLayoutTypes.KANBAN,
  EIssueLayoutTypes.CALENDAR,
];

type Props = {
  activeLayout: EIssueLayoutTypes | undefined;
  onChange: (layout: EIssueLayoutTypes) => void;
};

export function ParentIssueResponsiveLayoutSelection({ activeLayout, onChange }: Props) {
  return (
    <>
      <div className="hidden @4xl:flex">
        <LayoutSelection layouts={PARENT_ISSUE_LAYOUTS} onChange={onChange} selectedLayout={activeLayout} />
      </div>
      <div className="flex @4xl:hidden">
        <MobileLayoutSelection layouts={PARENT_ISSUE_LAYOUTS} onChange={onChange} activeLayout={activeLayout} />
      </div>
    </>
  );
}

export function persistParentIssueLayout(
  setDisplay: (patch: { layout: EIssueLayoutTypes }) => Promise<unknown>,
  layout: EIssueLayoutTypes
) {
  return setDisplay({ layout }).catch((error) => console.warn("Issue preferences could not be saved.", error));
}
