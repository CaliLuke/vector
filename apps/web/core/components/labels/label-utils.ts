/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IIssueLabel, TDropTarget } from "@plane/types";

/**
 * This provides a boolean to indicate if the label can be dropped onto the droptarget
 * @param source
 * @param label
 * @param isCurrentChild if the dropTarget is a child
 * @returns
 */
export const getCanDrop = (source: TDropTarget, label: IIssueLabel | undefined, isCurrentChild: boolean) => {
  const sourceData = source?.data;

  if (!sourceData) return false;

  // a label cannot be dropped on to itself and it's parent cannon be dropped on the child
  if (sourceData.id === label?.id || sourceData.id === label?.parent) return false;

  // if current dropTarget is a child and the label being dropped is a group then don't enable drop
  if (isCurrentChild && sourceData.isGroup) return false;

  return true;
};
