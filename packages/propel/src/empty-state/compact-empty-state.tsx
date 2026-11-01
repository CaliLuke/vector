/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { EmptyStateActions } from "./actions";
import { cn } from "@plane/utils/common";
import { getCompactAsset } from "./assets/asset-registry";
import type { BaseEmptyStateCommonProps } from "./types";

export function EmptyStateCompact({
  asset,
  assetKey,
  title,
  description,
  actions,
  className,
  rootClassName,
  assetClassName,
  align = "center",
  customButton,
}: BaseEmptyStateCommonProps) {
  // Determine which asset to use: assetKey takes precedence, fallback to custom asset
  const resolvedAsset = assetKey ? getCompactAsset(assetKey, assetClassName) : asset;

  const rootAlignClasses = align === "center" ? "items-center" : "items-start";
  const containerAlignClasses = align === "center" ? "items-center text-center" : "items-start text-left";

  return (
    <div className={cn("flex size-full items-center justify-center", rootAlignClasses, rootClassName)}>
      <div
        className={cn("flex size-full max-w-[25rem] flex-col justify-center gap-3", containerAlignClasses, className)}
      >
        {resolvedAsset && <div className="flex max-w-40 items-center">{resolvedAsset}</div>}

        <div className="flex flex-col gap-4">
          {title && description ? (
            <div className="flex flex-col gap-2">
              {title && <h3 className="text-16 leading-7 font-semibold text-primary">{title}</h3>}
              {description && <p className="text-13 leading-5 text-tertiary">{description}</p>}
            </div>
          ) : (
            title && <p className="text-13 leading-5 text-tertiary">{title}</p>
          )}

          <EmptyStateActions actions={actions} customButton={customButton} size="base" />
        </div>
      </div>
    </div>
  );
}

EmptyStateCompact.displayName = "EmptyStateCompact";
