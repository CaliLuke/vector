/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// local imports
import { EmptyStateActions } from "./actions";
import { cn } from "@plane/utils/common";
import { getDetailedAsset } from "./assets/asset-registry";
import type { BaseEmptyStateCommonProps } from "./types";

export function EmptyStateDetailed({
  asset,
  assetKey,
  title,
  description,
  actions,
  className,
  rootClassName,
  assetClassName,
  customButton,
  align = "start",
}: BaseEmptyStateCommonProps) {
  // Determine which asset to use: assetKey takes precedence, fallback to custom asset
  const resolvedAsset = assetKey ? getDetailedAsset(assetKey, assetClassName) : asset;

  return (
    <div className={cn("flex size-full items-center justify-center", rootClassName)}>
      <div
        className={cn(
          "flex size-full max-w-[25rem] flex-col justify-center gap-6 text-left",
          {
            "items-center text-center": align === "center",
          },
          className
        )}
      >
        {resolvedAsset && <div className="flex max-w-40 items-center">{resolvedAsset}</div>}

        <div
          className={cn("flex flex-col gap-4", {
            "items-center": align === "center",
          })}
        >
          {(title || description) && (
            <div className="flex flex-col gap-2">
              {title && <h3 className="text-16 leading-7 font-semibold text-primary">{title}</h3>}
              {description && <p className="text-13 leading-5 text-tertiary">{description}</p>}
            </div>
          )}

          <EmptyStateActions actions={actions} customButton={customButton} size="xl" />
        </div>
      </div>
    </div>
  );
}

EmptyStateDetailed.displayName = "EmptyStateDetailed";
