/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useContext } from "react";
import { PowerKRegistryContext } from "@/providers/native-command-state";

export function usePowerKCommandRegistry() {
  const context = useContext(PowerKRegistryContext);
  if (!context) throw new Error("usePowerKCommandRegistry must be used within NativeCommandStateProvider");
  return context;
}
