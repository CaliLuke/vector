/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import cloneDeep from "lodash-es/cloneDeep.js";
import isEqual from "lodash-es/isEqual.js";
import type {
  SingleOrArray,
  TAllAvailableOperatorsForDisplay,
  TFilterConditionForBuild,
  TFilterConditionNode,
  TFilterConditionNodeForDisplay,
  TFilterConditionPayload,
  TFilterExpression,
  TFilterProperty,
  TFilterValue,
  TLogicalOperator,
  TSupportedOperators,
} from "@plane/types";
import { EXTENDED_OPERATORS, FILTER_NODE_TYPE, LOGICAL_OPERATOR } from "@plane/types";
import {
  addAndCondition,
  createConditionNode,
  createNotGroupNode,
  createOrGroupNode,
  findNodeById,
  findConjunctiveCondition as findCondition,
  findParentChain,
  getDisplayOperator,
  getOperatorForPayload,
  hasChildrenProperty,
  hasValidValue,
  isConditionNode,
  isNotGroupNode,
  isOrGroupNode,
  removeNodeFromExpression,
  replaceNodeInExpression,
  shouldNotifyChangeForExpression,
  updateNodeInExpression,
} from "@plane/utils/rich-filters";

export function addFilterCondition<P extends TFilterProperty, V extends TFilterValue>(
  expression: TFilterExpression<P> | null,
  groupOperator: TLogicalOperator,
  condition: TFilterConditionPayload<P, V>,
  isNegation: boolean
): TFilterExpression<P> | null {
  const conditionNode = createConditionNode(condition);
  const nodeToAdd = isNegation ? createNotGroupNode(conditionNode) : conditionNode;
  if (groupOperator === LOGICAL_OPERATOR.AND) return addAndCondition(cloneDeep(expression), nodeToAdd);
  if (groupOperator === LOGICAL_OPERATOR.OR) {
    if (!expression) return nodeToAdd;
    const current = cloneDeep(expression);
    if (!isConditionNode(current) && isOrGroupNode(current)) {
      current.children.push(nodeToAdd);
      return current;
    }
    return createOrGroupNode([current, nodeToAdd]);
  }
  return expression;
}

export function updateFilterConditionProperty<P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  conditionId: string,
  property: P,
  operator: TSupportedOperators,
  isNegation: boolean
): TFilterExpression<P> {
  return updateFilterCondition(
    expression,
    conditionId,
    { property, operator, value: undefined, valueFormat: undefined },
    isNegation
  );
}

export function updateFilterConditionOperator<P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  conditionId: string,
  operator: TSupportedOperators,
  isNegation: boolean,
  shouldResetValue: boolean
): TFilterExpression<P> {
  return updateFilterCondition(
    expression,
    conditionId,
    shouldResetValue ? { operator, value: undefined } : { operator },
    isNegation
  );
}

export function updateFilterConditionValue<P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  conditionId: string,
  value: SingleOrArray<TFilterValue>,
  forceUpdate = false
): { expression: TFilterExpression<P> | null; changed: boolean } {
  const current = findNodeById(expression, conditionId);
  if (!current || current.type !== FILTER_NODE_TYPE.CONDITION) return { expression, changed: false };
  if (!hasValidValue(value)) {
    const result = removeNodeFromExpression(expression, conditionId);
    return { expression: result.expression, changed: result.shouldNotify };
  }
  if (!forceUpdate && isEqual(current.value, value)) return { expression, changed: false };
  const next = cloneDeep(expression);
  updateNodeInExpression(next, conditionId, { value });
  return { expression: next, changed: true };
}

export function removeFilterCondition<P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  conditionId: string
): { expression: TFilterExpression<P> | null; shouldNotify: boolean } {
  return removeNodeFromExpression(expression, conditionId);
}

export function updateFilterGroupOperator<P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  groupId: string,
  operator: "and" | "or"
): TFilterExpression<P> {
  const next = cloneDeep(expression);
  const group = findNodeById(next, groupId);
  if (group?.type === FILTER_NODE_TYPE.GROUP && hasChildrenProperty(group)) group.logicalOperator = operator;
  return next;
}

export function updateFilterCondition<P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  conditionId: string,
  payload: Partial<TFilterConditionNode<P, TFilterValue>>,
  isNegation: boolean
): TFilterExpression<P> {
  const next = cloneDeep(expression);
  updateNodeInExpression(next, conditionId, payload);
  const condition = findNodeById(next, conditionId);
  if (!condition || !isConditionNode(condition)) return next;
  const currentNegation = getOperatorForPayload(getDisplayOperator(condition.operator, next, conditionId)).isNegation;
  if (currentNegation === isNegation) return next;
  const parent = findParentChain(next, conditionId)?.[0];
  if (parent && isNotGroupNode(parent)) return replaceNodeInExpression(next, parent.id, condition);
  return replaceNodeInExpression(next, conditionId, createNotGroupNode(condition));
}

export function addSidebarFilterCondition<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  condition: TFilterConditionForBuild<P, TFilterValue>
): TFilterExpression<P> {
  const { operator, isNegation } = getOperatorForPayload(condition.operator);
  const payload = { property: condition.property, operator, value: condition.value };
  const existing = expression
    ? findConjunctiveCondition(expression, condition.property, condition.operator)
    : undefined;
  if (!expression || !existing)
    return addFilterCondition(expression, LOGICAL_OPERATOR.AND, payload, isNegation) ?? createConditionNode(payload);
  const updated = updateFilterConditionValue(expression, existing.id, condition.value, true);
  return updated.expression ?? createConditionNode(payload);
}

export function findConjunctiveCondition<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  property: P,
  operator: TAllAvailableOperatorsForDisplay
): TFilterConditionNodeForDisplay<P, TFilterValue> | undefined {
  if (!expression) return undefined;
  return findCondition(expression, property, operator);
}

export function isEmptyFilterExpression<P extends TFilterProperty>(expression: TFilterExpression<P> | null): boolean {
  return expression === null;
}

export function hasValidFilterValue(value: SingleOrArray<TFilterValue>): boolean {
  return hasValidValue(value);
}

export function shouldNotifyFilterChange<P extends TFilterProperty>(expression: TFilterExpression<P> | null): boolean {
  return shouldNotifyChangeForExpression(expression);
}

export function operatorValueMustReset<P extends TFilterProperty>(
  condition: TFilterConditionNode<P, TFilterValue>,
  currentConfigType: string | undefined,
  nextConfigType: string | undefined,
  operator: TSupportedOperators
): boolean {
  const changesPresenceCheck =
    (condition.operator === EXTENDED_OPERATORS.IS_NULL) !== (operator === EXTENDED_OPERATORS.IS_NULL);
  return changesPresenceCheck || currentConfigType !== nextConfigType;
}
