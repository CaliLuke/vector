/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isAxiosError } from "axios";

function responseErrorMessage(data: unknown): string | undefined {
  if (!data || typeof data !== "object" || !("message" in data) || typeof data.message !== "string") {
    return undefined;
  }
  return data.message;
}

/**
 * Application error class that sanitizes and standardizes errors across the app.
 * Extracts only essential information from AxiosError to prevent massive log bloat
 * and sensitive data leaks (cookies, tokens, etc).
 *
 * Usage:
 *   new AppError("Simple error message")
 *   new AppError("Custom error", { code: "MY_CODE", statusCode: 400 })
 *   new AppError(axiosError)  // Auto-extracts essential info
 *   new AppError(anyError)    // Works with any error type
 */
export class AppError extends Error {
  statusCode?: number;
  method?: string;
  url?: string;
  code?: string;
  context?: Record<string, unknown>;

  constructor(messageOrError: unknown, data?: Partial<Omit<AppError, "name" | "message">>) {
    // Handle error objects - extract essential info
    const error = messageOrError;

    // Already AppError - return immediately for performance (no need to re-process)
    if (error instanceof AppError) {
      return error;
    }

    // Handle string message (simple case like regular Error)
    if (typeof messageOrError === "string") {
      super(messageOrError);
      this.name = "AppError";
      if (data) {
        Object.assign(this, data);
      }
      return this;
    }

    // AxiosError - extract ONLY essential info (no config, no headers, no cookies)
    if (isAxiosError<unknown>(error)) {
      const axiosError = error;
      super(responseErrorMessage(axiosError.response?.data) || axiosError.message);
      this.name = "AppError";
      this.statusCode = axiosError.response?.status;
      this.method = axiosError.config?.method?.toUpperCase();
      this.url = axiosError.config?.url;
      this.code = axiosError.code;
      return this;
    }

    // DOMException (AbortError from cancelled requests)
    if (error instanceof DOMException && error.name === "AbortError") {
      super(error.message);
      this.name = "AppError";
      this.code = "ABORT_ERROR";
      return this;
    }

    // Standard Error objects
    if (error instanceof Error) {
      super(error.message);
      this.name = "AppError";
      this.code = error.name;
      return this;
    }

    // Unknown error types - safe fallback
    super("Unknown error occurred");
    this.name = "AppError";
    return this;
  }
}
