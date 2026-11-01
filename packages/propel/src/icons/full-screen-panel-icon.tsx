/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ISvgIcons } from "./type";
import { SquareIconFrame } from "./square-icon-frame";

export function FullScreenPanelIcon(props: ISvgIcons) {
  return (
    <SquareIconFrame {...props}>
      <path
        d="M16.6667 6.00001H7.20015C6.50001 5.99999 6.00005 6.00003 6.00017 7.33335C6.00027 8.4402 6.00021 13.8198 6.00018 15.8823L6.00017 16.6667C6.00017 18 6.00017 18 7.20015 18H16.6667C18 18 18 18 18 16.6667V7.33335C18 6.00001 18 6.00001 16.6667 6.00001H16.6667Z"
        fill="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </SquareIconFrame>
  );
}
