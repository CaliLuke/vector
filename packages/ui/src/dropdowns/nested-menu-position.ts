/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Options } from "@popperjs/core";

export const NESTED_MENU_POSITIONING: Partial<Options> = {
  strategy: "fixed",
  modifiers: [
    { name: "offset", options: { offset: [0, 4] } },
    {
      name: "flip",
      options: { fallbackPlacements: ["left-start", "right-end", "left-end", "top-start", "bottom-start"] },
    },
    { name: "preventOverflow", options: { padding: 8 } },
  ],
};
