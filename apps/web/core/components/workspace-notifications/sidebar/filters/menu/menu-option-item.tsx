/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ENotificationFilterType } from "@plane/constants";
import { CheckIcon } from "@plane/propel/icons";
// plane imports
// helpers
import { cn } from "@plane/utils/common";
// hooks
import { useNotifications } from "@/components/workspace-notifications/provider";

export const NotificationFilterOptionItem = function NotificationFilterOptionItem(props: {
  label: string;
  value: ENotificationFilterType;
}) {
  const { value, label } = props;
  // hooks
  const { filters, updateFilters } = useNotifications();

  const handleFilterTypeChange = (filterType: ENotificationFilterType, filterValue: boolean) =>
    updateFilters("type", {
      ...filters.type,
      [filterType]: filterValue,
    });

  // derived values
  const isSelected = filters?.type?.[value] || false;

  return (
    <div
      key={value}
      className="flex cursor-pointer items-center gap-2 rounded-xs p-1 px-2 transition-all hover:bg-layer-1"
      onClick={() => handleFilterTypeChange(value, !isSelected)}
    >
      <div
        className={cn("flex h-3 w-3 flex-shrink-0 items-center justify-center rounded-xs transition-all", {
          "bg-accent-primary text-on-color": isSelected,
          "bg-surface-2": !isSelected,
        })}
      >
        {isSelected && <CheckIcon className="h-2.5 w-2.5" />}
      </div>
      <div
        className={cn("text-body-xs-medium whitespace-nowrap", {
          "text-primary": isSelected,
          "text-secondary": !isSelected,
        })}
      >
        {label}
      </div>
    </div>
  );
};
