/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { useQuery } from "@tanstack/react-query";
import { useCurrentUser } from "./use-current-user";
import { currentUserSettingsOptions } from "@/queries/current-user-settings";

export function useCurrentUserSettings() {
  const { data: user } = useCurrentUser();
  return useQuery(currentUserSettingsOptions(user?.id));
}
