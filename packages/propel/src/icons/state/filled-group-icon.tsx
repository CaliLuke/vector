/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ISvgIcons } from "../type";
import { IconWrapper } from "../icon-wrapper";

export function FilledGroupIcon({
  path,
  color,
  width = "20",
  height = "20",
  className = "",
  ...rest
}: ISvgIcons & { path: string }) {
  return (
    <IconWrapper width={width} height={height} className={className} fill={undefined} {...rest}>
      <path d={path} fillRule="evenodd" clipRule="evenodd" fill={color} />
    </IconWrapper>
  );
}
