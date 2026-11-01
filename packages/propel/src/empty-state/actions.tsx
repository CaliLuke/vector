/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ComponentProps } from "react";
import { Button } from "../button/button";
import type { BaseEmptyStateCommonProps } from "./types";

export function EmptyStateActions({
  actions,
  customButton,
  size,
}: Pick<BaseEmptyStateCommonProps, "actions" | "customButton"> & { size: ComponentProps<typeof Button>["size"] }) {
  if (customButton) return customButton;
  if (!actions?.length) return null;
  return (
    <div className="flex flex-col gap-4 sm:flex-row">
      {actions.map(({ label, variant, ...props }) => (
        <Button key={props.id ?? label} variant={variant} size={size} {...props}>
          {label}
        </Button>
      ))}
    </div>
  );
}
