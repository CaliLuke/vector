/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type {
  TFilterAndGroupNode,
  TFilterOrGroupNode,
  TFilterNotGroupNode,
  TFilterExpression,
  TFilterGroupNode,
  TFilterProperty,
} from "@plane/types";
// local imports
import { getAndGroupChildren, isAndGroupNode, isNotGroupNode, isOrGroupNode } from "./core";

type TProcessGroupNodeHandlers<P extends TFilterProperty, T> = {
  onOrGroup: (group: TFilterOrGroupNode<P>) => T;
  onAndGroup: (group: TFilterAndGroupNode<P>) => T;
  onNotGroup: (group: TFilterNotGroupNode<P>) => T;
};

/**
 * Generic helper to process group nodes with type-safe handlers.
 * @param group - The group node to process
 * @param handlers - Object with handlers for each group type
 * @returns Result of the appropriate handler
 */
export const processGroupNode = <P extends TFilterProperty, T>(
  group: TFilterGroupNode<P>,
  handlers: TProcessGroupNodeHandlers<P, T>
): T => {
  if (isAndGroupNode(group)) {
    return handlers.onAndGroup(group);
  }
  if (isOrGroupNode(group)) return handlers.onOrGroup(group);
  if (isNotGroupNode(group)) return handlers.onNotGroup(group);
  throw new Error(`Invalid group node: unknown logical operator ${group}`);
};

/**
 * Gets the children of a group node, handling AND/OR groups (children array) and NOT groups (single child).
 * Uses processGroupNode for consistent group type handling.
 * @param group - The group node to get children from
 * @returns Array of child expressions
 */
export const getGroupChildren = <P extends TFilterProperty>(group: TFilterGroupNode<P>): TFilterExpression<P>[] =>
  processGroupNode(group, {
    onOrGroup: (orGroup) => orGroup.children,
    onAndGroup: (andGroup) => getAndGroupChildren(andGroup),
    onNotGroup: (notGroup) => [notGroup.child],
  });
