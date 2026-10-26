/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Tab } from "@headlessui/react";
// plane package imports
// plane web components
import TotalInsights from "../../total-insights";
import CreatedVsResolved from "../created-vs-resolved";
import CustomizedInsights from "../customized-insights";
import WorkItemsInsightTable from "../workitems-insight-table";

type Props = {
  fullScreen: boolean;
  isEpic?: boolean;
};

export const WorkItemsModalMainContent = function WorkItemsModalMainContent(props: Props) {
  const { fullScreen, isEpic } = props;

  return (
    <Tab.Group as={React.Fragment}>
      <div className="flex flex-col gap-14 overflow-y-auto p-6">
        <TotalInsights analyticsType="work-items" peekView={!fullScreen} />
        <CreatedVsResolved />
        <CustomizedInsights peekView={!fullScreen} isEpic={isEpic} />
        <WorkItemsInsightTable />
      </div>
    </Tab.Group>
  );
};
