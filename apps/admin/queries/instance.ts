/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { InstanceService, UserService } from "@plane/services";
import type { IFormattedInstanceConfiguration, IInstance } from "@plane/types";

const instanceService = new InstanceService();
const userService = new UserService();
export const instanceKeys = {
  info: ["instance", "info"] as const,
  user: ["instance", "user"] as const,
  admins: ["instance", "admins"] as const,
  configurations: ["instance", "configurations"] as const,
};

export const instanceInfoOptions = queryOptions({
  queryKey: instanceKeys.info,
  queryFn: ({ signal }) => instanceService.info(signal),
  staleTime: 30_000,
  retry: false,
});
export const currentUserOptions = queryOptions({
  queryKey: instanceKeys.user,
  queryFn: ({ signal }) => userService.adminDetails(signal),
  staleTime: 30_000,
  retry: false,
});
export const instanceAdminsOptions = queryOptions({
  queryKey: instanceKeys.admins,
  queryFn: ({ signal }) => instanceService.admins(signal),
  staleTime: 30_000,
});
export const instanceConfigurationsOptions = queryOptions({
  queryKey: instanceKeys.configurations,
  queryFn: ({ signal }) => instanceService.configurations(signal),
  staleTime: 30_000,
  select: (configurations) => {
    const values: Partial<IFormattedInstanceConfiguration> = {};
    for (const config of configurations) values[config.key] = config.value;
    return values;
  },
});

export function updateInstanceOptions(client: QueryClient) {
  return mutationOptions({
    mutationFn: (data: Partial<IInstance>) => instanceService.update(data),
    onSuccess: async (instance) => {
      await client.cancelQueries({ queryKey: instanceKeys.info });
      client.setQueryData(instanceInfoOptions.queryKey, (info) => (info ? { ...info, instance } : info));
      await client.invalidateQueries({ queryKey: instanceKeys.info });
    },
  });
}
export function updateConfigurationsOptions(client: QueryClient) {
  return mutationOptions({
    mutationKey: instanceKeys.configurations,
    scope: { id: "instance-configurations" },
    mutationFn: (data: Partial<IFormattedInstanceConfiguration>) => instanceService.updateConfigurations(data),
    onSuccess: async (updated) => {
      await client.cancelQueries({ queryKey: instanceKeys.configurations });
      client.setQueryData(instanceConfigurationsOptions.queryKey, (previous) => {
        if (!previous) return undefined;
        const entries = new Map(previous.map((config) => [config.key, config]));
        for (const config of updated) entries.set(config.key, config);
        return [...entries.values()];
      });
      await client.invalidateQueries({ queryKey: instanceKeys.configurations });
    },
  });
}
export function disableEmailOptions(client: QueryClient) {
  return mutationOptions({
    mutationKey: instanceKeys.configurations,
    scope: { id: "instance-configurations" },
    mutationFn: () => instanceService.disableEmail(),
    onSuccess: () => client.invalidateQueries({ queryKey: instanceKeys.configurations }),
  });
}

export const sendTestEmailOptions = mutationOptions({
  mutationFn: (receiver: string) => instanceService.sendTestEmail(receiver),
});
