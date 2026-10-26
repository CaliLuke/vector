/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { setPromiseToast } from "@plane/propel/toast";

export function notifyConfigurationSave(request: Promise<unknown>) {
  setPromiseToast(request, {
    loading: "Saving configuration",
    success: { title: "Success", message: () => "Configuration saved successfully" },
    error: { title: "Error", message: () => "Failed to save configuration" },
  });
  void request.catch(console.error);
}
