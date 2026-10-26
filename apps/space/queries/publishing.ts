/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { queryOptions, skipToken } from "@tanstack/react-query";
import { SitesProjectPublishService } from "@plane/services";

const service = new SitesProjectPublishService();
export function publishSettingsOptions(anchor: string | undefined) {
  return queryOptions({
    queryKey: ["public", anchor, "settings"],
    queryFn: anchor ? ({ signal }) => service.retrieveSettingsByAnchor(anchor, signal) : skipToken,
    staleTime: 30_000,
  });
}
