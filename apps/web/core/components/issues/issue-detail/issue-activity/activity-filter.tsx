/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ListFilter } from "lucide-react";
// plane imports
import type { TActivityFilters, TActivityFilterOption } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { IconButton } from "@plane/propel/icon-button";
import { CheckIcon } from "@plane/propel/icons";
import { PopoverMenu } from "@plane/ui/popovers";
// helper
import { cn } from "@plane/utils/common";
// constants

type TActivityFilter = {
  selectedFilters: TActivityFilters[];
  filterOptions: TActivityFilterOption[];
};

export const ActivityFilter = function ActivityFilter(props: TActivityFilter) {
  const { selectedFilters = [], filterOptions } = props;

  // hooks
  const { t } = useTranslation();

  return (
    <PopoverMenu
      buttonClassName="outline-none"
      button={
        <IconButton
          variant="tertiary"
          icon={ListFilter}
          className={
            selectedFilters.length < filterOptions.length
              ? "relative after:absolute after:-top-0.5 after:-right-0.5 after:size-2 after:rounded-full after:bg-accent-primary after:content-['']"
              : undefined
          }
        />
      }
      buttonAsChild
      panelClassName="p-2 rounded-md border border-subtle bg-surface-1"
      data={filterOptions}
      keyExtractor={(item) => item.key}
      render={(item) => (
        <button
          key={item.key}
          type="button"
          aria-pressed={item.isSelected}
          className="flex w-full cursor-pointer items-center gap-2 rounded-xs p-1 px-2 text-left text-13 transition-all hover:bg-layer-1"
          onClick={item.onClick}
        >
          <span
            className={cn(
              "flex h-3 w-3 flex-shrink-0 items-center justify-center rounded-xs bg-surface-2 transition-all",
              {
                "bg-accent-primary text-on-color": item.isSelected,
                "bg-layer-1 text-placeholder": item.isSelected && selectedFilters.length === 1,
                "bg-surface-2": !item.isSelected,
              }
            )}
          >
            {item.isSelected && <CheckIcon className="h-2.5 w-2.5" />}
          </span>
          <span className={cn("whitespace-nowrap", item.isSelected ? "text-primary" : "text-secondary")}>
            {t(item.labelTranslationKey)}
          </span>
        </button>
      )}
    />
  );
};
