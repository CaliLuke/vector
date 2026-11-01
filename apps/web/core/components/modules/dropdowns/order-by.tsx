import { OrderByMenuItems } from "@/components/common/order-by-menu-items";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ArrowDownWideNarrow, ArrowUpWideNarrow } from "lucide-react";
import { MODULE_ORDER_BY_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { getButtonStyling } from "@plane/propel/button";
import { ChevronDownIcon } from "@plane/propel/icons";
import type { TModuleOrderByOptions } from "@plane/types";
// ui
import { CustomMenu } from "@plane/ui";
// helpers
import { cn } from "@plane/utils/common";
// types
// constants

type Props = {
  onChange: (value: TModuleOrderByOptions) => void;
  value: TModuleOrderByOptions | undefined;
};

export function ModuleOrderByDropdown(props: Props) {
  const { onChange, value } = props;
  // hooks
  const { t } = useTranslation();

  const orderByDetails = MODULE_ORDER_BY_OPTIONS.find(
    (option) => option.key === value || (option.descendingKey !== undefined && option.descendingKey === value)
  );

  const isDescending = value?.[0] === "-";

  return (
    <CustomMenu
      customButton={
        <div className={cn(getButtonStyling("secondary", "lg"), "px-2 text-tertiary")}>
          {!isDescending ? <ArrowUpWideNarrow className="size-3" /> : <ArrowDownWideNarrow className="size-3" />}
          {orderByDetails && t(orderByDetails?.i18n_label)}
          <ChevronDownIcon className="size-3" strokeWidth={2} />
        </div>
      }
      placement="bottom-end"
      maxHeight="lg"
      closeOnSelect
    >
      <OrderByMenuItems options={MODULE_ORDER_BY_OPTIONS} value={value} onChange={onChange} hideManualDirections />
    </CustomMenu>
  );
}
