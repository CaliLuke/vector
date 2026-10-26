import type {
  TAllAvailableOperatorsForDisplay,
  TFilterConditionNodeForDisplay,
  TFilterExpression,
  TFilterProperty,
  TFilterValue,
} from "@plane/types";
import { isAndGroupNode, isConditionNode, isNotGroupNode } from "../../types/core";

/** Sidebar choices describe restrictions on every result, not scoped OR branches. */
export function findConjunctiveCondition<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  property: P,
  operator: TAllAvailableOperatorsForDisplay
): TFilterConditionNodeForDisplay<P, TFilterValue> | undefined {
  if (!expression) return undefined;
  let node = expression;
  let excluded = false;
  while (isNotGroupNode(node)) {
    excluded = !excluded;
    node = node.child;
  }
  if (isConditionNode(node)) {
    const displayOperator: TAllAvailableOperatorsForDisplay = excluded ? `not_${node.operator}` : node.operator;
    return node.property === property && displayOperator === operator
      ? { ...node, operator: displayOperator }
      : undefined;
  }
  if (isNotGroupNode(expression) || !isAndGroupNode(node)) return undefined;
  for (const child of node.children) {
    const match = findConjunctiveCondition(child, property, operator);
    if (match) return match;
  }
  return undefined;
}
