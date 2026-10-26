/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { CloseIcon } from "@plane/propel/icons";
import type { TFilterProperty } from "@plane/types";
import type { TFilterExpressionEditor } from "../shared";

interface FilterItemCloseButtonProps<P extends TFilterProperty> {
  conditionId: string;
  editor: TFilterExpressionEditor<P>;
}

export const FilterItemCloseButton = function FilterItemCloseButton<P extends TFilterProperty>(
  props: FilterItemCloseButtonProps<P>
) {
  const { conditionId, editor } = props;

  const handleRemoveFilter = () => {
    editor.onRemoveCondition(conditionId);
  };

  return (
    <button
      onClick={handleRemoveFilter}
      className="bg-layer-transparent px-1.5 text-placeholder hover:bg-layer-transparent-hover hover:text-tertiary focus:outline-none"
      type="button"
      aria-label="Remove filter"
    >
      <CloseIcon className="size-3.5" />
    </button>
  );
};
