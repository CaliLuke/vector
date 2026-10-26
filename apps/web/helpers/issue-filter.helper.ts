/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import type { IIssueDisplayProperties } from "@plane/types";
// lib
export const shouldRenderColumn = (key: keyof IIssueDisplayProperties, isEstimateEnabled: boolean): boolean => {
  switch (key) {
    case "estimate":
      return isEstimateEnabled;
    default:
      return true;
  }
};
