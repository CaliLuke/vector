/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { AxiosInstance } from "axios";
import { APIService as SharedAPIService } from "@plane/services";

export function installUnauthorizedRedirectInterceptor(axiosInstance: AxiosInstance) {
  axiosInstance.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.response && error.response.status === 401) {
        const { pathname, search, hash } = window.location;
        // A late request must not replace the destination already held by sign-in.
        if (pathname !== "/") {
          const query = new URLSearchParams({ next_path: `${pathname}${search}${hash}` });
          window.location.replace(`/?${query}`);
        }
      }
      return Promise.reject(error);
    }
  );
}

export abstract class APIService extends SharedAPIService {
  constructor(baseURL: string) {
    super(baseURL, installUnauthorizedRedirectInterceptor);
  }
}
