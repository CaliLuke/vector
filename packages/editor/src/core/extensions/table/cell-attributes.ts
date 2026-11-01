/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DEFAULT_COLUMN_WIDTH } from "./table";

export function cellGeometryAttributes() {
  return {
    colspan: { default: 1 },
    rowspan: { default: 1 },
    colwidth: {
      default: [DEFAULT_COLUMN_WIDTH],
      parseHTML: (element: HTMLElement) => {
        const width = element.getAttribute("colwidth");
        return width ? [parseInt(width, 10)] : null;
      },
    },
  };
}
