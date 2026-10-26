/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ListFilter } from "lucide-react";
// plane imports
import type { ENotificationFilterType } from "@plane/constants";
import { FILTER_TYPE_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { PopoverMenu } from "@plane/ui";
// local imports
import { NotificationFilterOptionItem } from "./menu-option-item";
import { IconButton } from "@plane/propel/icon-button";

export const NotificationFilter = function NotificationFilter() {
  const { t } = useTranslation();

  // oxlint-disable-next-line no-map-spread -- Translations must not mutate shared filter options.
  const translatedFilterTypeOptions = FILTER_TYPE_OPTIONS.map((filter) => ({
    ...filter,
    label: t(filter.i18n_label),
  }));

  return (
    <PopoverMenu
      data={translatedFilterTypeOptions}
      button={
        <IconButton
          size="base"
          variant="ghost"
          icon={ListFilter}
          aria-label={t("notification.options.filters")}
          title={t("notification.options.filters")}
        />
      }
      buttonAsChild
      keyExtractor={(item: { label: string; value: ENotificationFilterType }) => item.value}
      render={(item) => <NotificationFilterOptionItem {...item} />}
    />
  );
};
