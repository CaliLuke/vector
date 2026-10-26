/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { cn } from "@plane/utils/common";
// local imports
import type { TPopoverMenuOptions } from "./root";

type Props = Omit<TPopoverMenuOptions, "key">;

export const NotificationMenuOptionItem = function NotificationMenuOptionItem(props: Props) {
  const { type, label = "", isActive, prependIcon, appendIcon, onClick } = props;

  if (type === "menu-item")
    return (
      <button
        type="button"
        aria-pressed={Boolean(isActive)}
        className="mx-2 flex cursor-pointer items-center gap-2 rounded-xs p-1 px-2 transition-all hover:bg-layer-1"
        onClick={() => onClick && onClick()}
      >
        {prependIcon}
        <span
          className={cn("text-body-xs-medium whitespace-nowrap", {
            "text-primary": isActive,
            "text-secondary": !isActive,
          })}
        >
          {label}
        </span>
        {appendIcon && <span className="ml-auto">{appendIcon}</span>}
      </button>
    );

  return <div className="border-b border-subtle" />;
};
