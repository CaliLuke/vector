/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
export function getAiErrorMessage(error: unknown): string {
  if (error && typeof error === "object") {
    if (
      "data" in error &&
      error.data &&
      typeof error.data === "object" &&
      "error" in error.data &&
      typeof error.data.error === "string"
    )
      return error.data.error;
    if ("status" in error && error.status === 429)
      return "You have reached the maximum number of requests of 50 requests per month per user.";
  }
  return "Some error occurred. Please try again.";
}
