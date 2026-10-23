/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IFavorite, TDropTarget } from "@plane/types";

/**
 * This provides a boolean to indicate if the favorite can be dropped onto the droptarget
 * @param source
 * @param favorite
 * @param isCurrentChild if the dropTarget is a child
 * @returns
 */
export const getCanDrop = (source: TDropTarget, favorite: IFavorite | undefined, isCurrentChild: boolean) => {
  const sourceData = source?.data;

  if (!sourceData) return false;

  // a favorite cannot be dropped on to itself
  if (sourceData.id === favorite?.id) return false;

  // if current dropTarget is a child and the favorite being dropped is a group then don't enable drop
  if (isCurrentChild && sourceData.isGroup) return false;

  return true;
};
