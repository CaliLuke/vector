/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TChartMargin } from "@plane/types";

export function chartMargin({ top = 5, right = 30, bottom = 5, left = 20 }: TChartMargin = {}) {
  return { top, right, bottom, left };
}
