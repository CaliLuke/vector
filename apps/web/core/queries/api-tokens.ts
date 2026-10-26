/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { QueryClient } from "@tanstack/react-query";
import { queryOptions, mutationOptions } from "@tanstack/react-query";
import { API_TOKENS_LIST } from "@plane/constants";
import type { IApiToken } from "@plane/types";
import { APITokenService } from "@plane/services";

const apiTokenService = new APITokenService();

export const apiTokensOptions = queryOptions({
  queryKey: API_TOKENS_LIST,
  queryFn: ({ signal }) => apiTokenService.list(signal),
});

export function createApiTokenOptions(client: QueryClient) {
  const key = apiTokensOptions.queryKey;
  return mutationOptions({
    mutationFn: (data: Partial<IApiToken>) => apiTokenService.create(data),
    retry: false,
    onMutate: () => client.getQueryCache().find({ queryKey: key, exact: true }),
    onSuccess: async (createdToken, _data, query) => {
      if (!query || client.getQueryCache().find({ queryKey: key, exact: true }) !== query) return;
      await client.cancelQueries({ queryKey: key });
      if (client.getQueryCache().find({ queryKey: key, exact: true }) !== query) return;

      const listToken = { ...createdToken };
      delete listToken.token;
      client.setQueryData<IApiToken[]>(key, (tokens) => (tokens ? [listToken, ...tokens] : tokens));
      await client.invalidateQueries({ queryKey: key });
    },
  });
}

export function deleteApiTokenOptions(client: QueryClient) {
  const key = apiTokensOptions.queryKey;
  return mutationOptions({
    mutationFn: (tokenId: string) => apiTokenService.destroy(tokenId),
    retry: false,
    onMutate: () => client.getQueryCache().find({ queryKey: key, exact: true }),
    onSuccess: async (_deleted, tokenId, query) => {
      if (!query || client.getQueryCache().find({ queryKey: key, exact: true }) !== query) return;
      await client.cancelQueries({ queryKey: key });
      if (client.getQueryCache().find({ queryKey: key, exact: true }) !== query) return;

      client.setQueryData<IApiToken[]>(key, (tokens) => tokens?.filter((token) => token.id !== tokenId));
      await client.invalidateQueries({ queryKey: key });
    },
  });
}
