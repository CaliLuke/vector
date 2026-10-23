/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";

import { useLocation } from "@tanstack/react-router";

type TParamsToAdd = {
  [key: string]: string;
};

export const useQueryParams = () => {
  // URL state
  const search = useLocation({ select: (location) => location.searchStr });
  const pathname = useLocation({ select: (location) => location.pathname });

  const updateQueryParams = useCallback(
    ({ paramsToAdd = {}, paramsToRemove = [] }: { paramsToAdd?: TParamsToAdd; paramsToRemove?: string[] }) => {
      const currentParams = new URLSearchParams(search);

      // add or update query parameters
      Object.keys(paramsToAdd).forEach((key) => {
        currentParams.set(key, paramsToAdd[key]);
      });

      // remove specified query parameters
      paramsToRemove.forEach((key) => {
        currentParams.delete(key);
      });

      // construct the new route with the updated query parameters
      const query = currentParams.toString();
      const newRoute = query ? `${pathname}?${query}` : pathname;
      return newRoute;
    },
    [pathname, search]
  );

  return {
    updateQueryParams,
  };
};
