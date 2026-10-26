/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import type { TDropdownProps } from "../types";

export type StateDropdownProps = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  iconSize?: string;
  onChange: (val: string) => void;
  onClose?: () => void;
  projectId: string | undefined;
  renderByDefault?: boolean;
  showDefaultState?: boolean;
  stateIds?: string[];
  value: string | undefined | null;
};
