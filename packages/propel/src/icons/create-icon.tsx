/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ISvgIcons } from "./type";
import { SquareIconFrame } from "./square-icon-frame";

export function CreateIcon(props: ISvgIcons) {
  return (
    <SquareIconFrame openCorner {...props}>
      <path d="M16 5H22" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M19 2V8" strokeLinecap="round" strokeLinejoin="round" />
    </SquareIconFrame>
  );
}
