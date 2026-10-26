/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { v4 as uuidv4 } from "uuid";
// plane imports
import type {
  TFilterAndGroupNode,
  TFilterOrGroupNode,
  TFilterNotGroupNode,
  TFilterConditionNode,
  TFilterConditionPayload,
  TFilterExpression,
  TFilterProperty,
  TFilterValue,
} from "@plane/types";
import { FILTER_NODE_TYPE, LOGICAL_OPERATOR } from "@plane/types";

/**
 * Creates a condition node with a unique ID.
 * @param condition - The condition to create
 * @returns The created condition node
 */
export const createConditionNode = <P extends TFilterProperty, V extends TFilterValue>(
  condition: TFilterConditionPayload<P, V>
): TFilterConditionNode<P, V> => ({
  id: uuidv4(),
  type: FILTER_NODE_TYPE.CONDITION,
  ...condition,
});

/**
 * Creates an AND group node with a unique ID.
 * @param nodes - The nodes to add to the group
 * @returns The created AND group node
 */
export const createAndGroupNode = <P extends TFilterProperty>(
  nodes: TFilterExpression<P>[]
): TFilterAndGroupNode<P> => ({
  id: uuidv4(),
  type: FILTER_NODE_TYPE.GROUP,
  logicalOperator: LOGICAL_OPERATOR.AND,
  children: nodes,
});

export const createNotGroupNode = <P extends TFilterProperty>(child: TFilterExpression<P>): TFilterNotGroupNode<P> => ({
  id: uuidv4(),
  type: FILTER_NODE_TYPE.GROUP,
  logicalOperator: LOGICAL_OPERATOR.NOT,
  child,
});

export const createOrGroupNode = <P extends TFilterProperty>(nodes: TFilterExpression<P>[]): TFilterOrGroupNode<P> => ({
  id: uuidv4(),
  type: FILTER_NODE_TYPE.GROUP,
  logicalOperator: LOGICAL_OPERATOR.OR,
  children: nodes,
});
