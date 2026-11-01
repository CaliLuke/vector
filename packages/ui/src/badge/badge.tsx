/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ButtonBaseProps } from "../button/base";
import { createButton } from "../button/base";
import type { TBadgeVariant, TBadgeSizes } from "./helper";
import { getIconStyling, getBadgeStyling } from "./helper";

export interface BadgeProps extends ButtonBaseProps {
  variant?: TBadgeVariant;
  size?: TBadgeSizes;
}

export const Badge = createButton<TBadgeVariant, TBadgeSizes>({
  displayName: "plane-ui-badge",
  defaultVariant: "primary",
  defaultSize: "md",
  getStyle: getBadgeStyling,
  getIconStyle: getIconStyling,
});
