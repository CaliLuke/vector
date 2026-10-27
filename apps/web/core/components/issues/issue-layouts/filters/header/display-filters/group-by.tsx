/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ParentTypeGrouping } from "./parent-type-grouping";
import { CustomPropertyGrouping } from "./custom-property-grouping";
import React, { useState } from "react";

import { useTranslation } from "@plane/i18n";
import type { IIssueDisplayFilterOptions, TIssueGroupByOptions } from "@plane/types";
// helpers
import { useGroupByOptions } from "../../../utils";
// components
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";

type Props = {
  isEpic?: boolean;
  displayFilters: IIssueDisplayFilterOptions | undefined;
  groupByOptions: TIssueGroupByOptions[];
  handleUpdate: (val: TIssueGroupByOptions) => void;
  ignoreGroupedFilters: Partial<TIssueGroupByOptions>[];
};

export const FilterGroupBy = function FilterGroupBy(props: Props) {
  const { displayFilters, groupByOptions, handleUpdate, ignoreGroupedFilters } = props;
  // hooks
  const { t } = useTranslation();
  const [previewEnabled, setPreviewEnabled] = useState(true);

  const selectedGroupBy = displayFilters?.group_by ?? null;
  const selectedSubGroupBy = displayFilters?.sub_group_by ?? null;

  const options = useGroupByOptions(groupByOptions);

  return (
    <>
      <FilterHeader
        title={t("common.group_by")}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {options.map((groupBy) => {
            if (
              displayFilters?.layout === "kanban" &&
              selectedSubGroupBy !== null &&
              groupBy.key === selectedSubGroupBy
            )
              return null;
            if (ignoreGroupedFilters.includes(groupBy?.key)) return null;

            return (
              <FilterOption
                key={groupBy?.key}
                isChecked={selectedGroupBy === groupBy?.key}
                onClick={() => handleUpdate(groupBy.key)}
                title={t(groupBy.titleTranslationKey)}
                multiple={false}
              />
            );
          })}
          <ParentTypeGrouping displayFilters={displayFilters} isEpic={props.isEpic} onChange={handleUpdate} />
          <CustomPropertyGrouping
            displayFilters={displayFilters}
            axis="group_by"
            isEpic={props.isEpic}
            onChange={handleUpdate}
          />
        </div>
      )}
    </>
  );
};
