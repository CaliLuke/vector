/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ISvgIcons } from "./type";
import { SquareIconFrame } from "./square-icon-frame";

export function SidePanelIcon(props: ISvgIcons) {
  return (
    <SquareIconFrame {...props}>
      <path
        d="M20 3H12V5V19V21H20C20.5523 21 21 20.1046 21 19V5C21 3.89543 20.5523 3 20 3Z"
        fill="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </SquareIconFrame>
  );
}
