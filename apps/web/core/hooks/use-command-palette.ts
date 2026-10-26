/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useContext } from "react";
import { CommandPaletteContext } from "@/providers/native-command-state";

export function useCommandPalette() {
  const context = useContext(CommandPaletteContext);
  if (!context) throw new Error("useCommandPalette must be used within NativeCommandStateProvider");
  return context;
}
