/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { queryOptions } from "@tanstack/react-query";
import { InstanceService, UserService } from "@plane/services";

const instanceService = new InstanceService();
const userService = new UserService();
export const instanceInfoOptions = queryOptions({
  queryKey: ["public", "instance"],
  queryFn: ({ signal }) => instanceService.info(signal),
  staleTime: 30_000,
  refetchOnWindowFocus: false,
  retry: false,
});
export const currentUserOptions = queryOptions({
  queryKey: ["public", "current-user"],
  queryFn: ({ signal }) => userService.me(signal),
  staleTime: 30_000,
  retry: false,
});
