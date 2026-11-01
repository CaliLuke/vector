/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { AxiosInstance, AxiosRequestConfig } from "axios";
import { create } from "axios";
import { env } from "@/env";
import { AppError } from "@/lib/errors";

export abstract class APIService {
  protected baseURL: string;
  private axiosInstance: AxiosInstance;
  private header: Record<string, string> = {};

  constructor(baseURL?: string) {
    this.baseURL = baseURL || env.API_BASE_URL;
    this.axiosInstance = create({
      baseURL: this.baseURL,
      withCredentials: true,
      timeout: 20000,
    });
    this.setupInterceptors();
  }

  private setupInterceptors() {
    this.axiosInstance.interceptors.response.use(
      (response) => response,
      (error) => {
        return Promise.reject(new AppError(error));
      }
    );
  }

  setHeader(key: string, value: string) {
    this.header[key] = value;
  }

  getHeader() {
    return this.header;
  }

  get<T = unknown>(url: string, params: AxiosRequestConfig = {}, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.get<T>(url, {
      ...params,
      ...config,
    });
  }

  post<T = unknown>(url: string, data: unknown = {}, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.post<T>(url, data, config);
  }

  put<T = unknown>(url: string, data: unknown = {}, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.put<T>(url, data, config);
  }

  patch<T = unknown>(url: string, data: unknown = {}, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.patch<T>(url, data, config);
  }

  delete<T = unknown>(url: string, data?: Record<string, unknown> | null | string, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.delete<T>(url, { data, ...config });
  }

  request<T = unknown>(config: AxiosRequestConfig = {}) {
    return this.axiosInstance.request<T>(config);
  }
}
