/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TDropTarget, TWidgetEntityData } from "@plane/types";

/**
 * This provides a boolean to indicate if the widget can be dropped onto the droptarget
 * @param source
 * @param widget
 * @returns
 */
export const getCanDrop = (source: TDropTarget, widget: TWidgetEntityData | undefined) => {
  const sourceData = source?.data;

  if (!sourceData) return false;

  // a widget cannot be dropped on to itself
  if (sourceData.id === widget?.key) return false;

  return true;
};
