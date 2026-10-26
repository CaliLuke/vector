/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  SingleOrArray,
  TFilterConditionNode,
  TFilterConditionNodeForDisplay,
  TFilterConfig,
  TFilterExpression,
  TFilterProperty,
  TFilterValue,
  TLogicalOperator,
  TSupportedOperators,
  TSupportedFilterFieldConfigs,
} from "@plane/types";

export const COMMON_FILTER_ITEM_BORDER_CLASSNAME = "border-r border-subtle-1";

export const EMPTY_FILTER_PLACEHOLDER_TEXT = "--";

export type TFilterValueInputProps<P extends TFilterProperty, V extends TFilterValue> = {
  condition: TFilterConditionNodeForDisplay<P, V>;
  filterFieldConfig: TSupportedFilterFieldConfigs<V>;
  isDisabled?: boolean;
  onChange: (values: SingleOrArray<V>) => void;
};

export type TFilterExpressionEditor<P extends TFilterProperty> = {
  expression: TFilterExpression<P> | null;
  conditions: TFilterConditionNode<P, TFilterValue>[];
  displayConditions: TFilterConditionNodeForDisplay<P, TFilterValue>[];
  configs: TFilterConfig<P>[];
  isConfigReady: boolean;
  conversionError: string | null;
  editingError: string | null;
  hasActiveFilters: boolean;
  hasChanges: boolean;
  isVisible: boolean;
  clearLabel?: string;
  saveLabel?: string;
  updateLabel?: string;
  canClear: boolean;
  canSave: boolean;
  canUpdate: boolean;
  onToggleVisibility: (visible?: boolean) => void;
  onAddCondition: (property: P, operator: TSupportedOperators, isNegation: boolean) => void;
  onUpdateConditionProperty: (
    conditionId: string,
    property: P,
    operator: TSupportedOperators,
    isNegation: boolean
  ) => void;
  onUpdateConditionOperator: (conditionId: string, operator: TSupportedOperators, isNegation: boolean) => void;
  onUpdateConditionValue: (conditionId: string, value: SingleOrArray<TFilterValue>) => void;
  onRemoveCondition: (conditionId: string) => void;
  onUpdateGroupOperator: (groupId: string, operator: Exclude<TLogicalOperator, "not">) => void;
  onClear: () => Promise<void>;
  onSave: () => Promise<void>;
  onUpdate: () => Promise<void>;
};
