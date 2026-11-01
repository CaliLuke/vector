/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export const emptyStateCommonArgTypes = {
  className: {
    control: "text",
    description: "Additional CSS classes to apply to the content wrapper",
  },
  rootClassName: {
    control: "text",
    description: "Additional CSS classes to apply to the root container",
  },
  assetClassName: {
    control: "text",
    description: "Additional CSS classes to apply to the asset",
  },
  asset: {
    control: false,
    description: "Custom React node to display as the visual asset (use this for full control instead of assetKey)",
  },
  actions: {
    control: false,
    description: "Array of action buttons to display",
  },
} as const;
