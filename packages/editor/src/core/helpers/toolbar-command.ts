/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ToolbarMenuItem } from "@/constants/common";
import type { EditorRefApi } from "@/types";

export function executeToolbarCommand(execute: EditorRefApi["executeMenuItemCommand"], item: ToolbarMenuItem) {
  switch (item.itemKey) {
    case "image":
    case "attachment":
      execute({ itemKey: item.itemKey, savedSelection: item.extraProps.savedSelection });
      break;
    case "text-color":
    case "background-color":
      execute({ itemKey: item.itemKey, color: item.extraProps.color });
      break;
    case "link":
      execute({ itemKey: item.itemKey, ...item.extraProps });
      break;
    case "text-align":
      execute({ itemKey: item.itemKey, alignment: item.extraProps.alignment });
      break;
    default:
      execute({ itemKey: item.itemKey });
  }
}
