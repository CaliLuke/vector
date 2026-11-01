/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ButtonBaseProps } from "./base";
import { createButton } from "./base";
import type { TButtonVariant, TButtonSizes } from "./helper";
import { getIconStyling, getButtonStyling } from "./helper";

export interface ButtonProps extends ButtonBaseProps {
  variant?: TButtonVariant;
  size?: TButtonSizes;
}

export const Button = createButton<TButtonVariant, TButtonSizes>({
  displayName: "plane-ui-button",
  defaultVariant: "primary",
  defaultSize: "md",
  getStyle: getButtonStyling,
  getIconStyle: getIconStyling,
});
