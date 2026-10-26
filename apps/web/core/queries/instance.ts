/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { queryOptions } from "@tanstack/react-query";
import { InstanceService } from "@plane/services";

const service = new InstanceService();

export const instanceInfoOptions = queryOptions({
  queryKey: ["instance", "info"],
  queryFn: ({ signal }) => service.info(signal),
  staleTime: 30_000,
  refetchOnMount: true,
  refetchOnWindowFocus: false,
});

export const instanceConfigOptions = queryOptions({
  ...instanceInfoOptions,
  select: (info) => info.config,
});
