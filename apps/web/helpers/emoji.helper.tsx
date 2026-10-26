/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TLogoProps } from "@plane/types";
import type { TChangeHandlerProps } from "@plane/propel/emoji-icon-picker";

export function logoPropsFromSelection(selection: TChangeHandlerProps): TLogoProps {
  return selection.type === "emoji"
    ? { in_use: "emoji", emoji: { value: selection.value } }
    : { in_use: "icon", icon: selection.value };
}
