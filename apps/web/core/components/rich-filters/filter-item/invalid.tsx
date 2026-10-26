/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { CircleAlert } from "lucide-react";
// plane imports
import type { TFilterProperty } from "@plane/types";
// local imports
import { FilterItemCloseButton } from "./close-button";
import { FilterItemContainer } from "./container";
import { FilterItemProperty } from "./property";
import type { IFilterItemProps } from "./root";

export const InvalidFilterItem = function InvalidFilterItem<P extends TFilterProperty>(props: IFilterItemProps<P>) {
  const { condition, editor, isDisabled = false, showTransition = true } = props;

  return (
    <FilterItemContainer
      conditionValue={condition.value}
      showTransition={showTransition}
      variant="error"
      tooltipContent="This filter condition is no longer valid. The property may have been deleted or your access to it may have changed."
    >
      {/* Property section */}
      <FilterItemProperty
        conditionId={condition.id}
        icon={CircleAlert}
        label="Invalid filter"
        editor={editor}
        isDisabled={isDisabled}
      />
      <span className="max-w-64 truncate px-2 text-12" title={JSON.stringify(condition.value)}>
        {condition.property} {condition.operator}: {JSON.stringify(condition.value)}
      </span>
      {/* Remove button */}
      {!isDisabled && <FilterItemCloseButton conditionId={condition.id} editor={editor} />}
    </FilterItemContainer>
  );
};
