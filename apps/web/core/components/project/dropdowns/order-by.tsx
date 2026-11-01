import { OrderByMenuItems } from "@/components/common/order-by-menu-items";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ArrowDownWideNarrow } from "lucide-react";
// plane imports
import { PROJECT_ORDER_BY_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { getButtonStyling } from "@plane/propel/button";
import type { TProjectOrderByOptions } from "@plane/types";
import { CustomMenu } from "@plane/ui";

type Props = {
  onChange: (value: TProjectOrderByOptions) => void;
  value: TProjectOrderByOptions | undefined;
  isMobile?: boolean;
};

export function ProjectOrderByDropdown(props: Props) {
  const { onChange, value, isMobile = false } = props;
  const { t } = useTranslation();

  const orderByDetails = PROJECT_ORDER_BY_OPTIONS.find(
    (option) => option.key === value || (option.descendingKey !== undefined && option.descendingKey === value)
  );

  return (
    <CustomMenu
      className={isMobile ? "flex w-full justify-center" : ""}
      customButton={
        <div className={getButtonStyling("secondary", "lg")}>
          <ArrowDownWideNarrow className="size-3.5 shrink-0" strokeWidth={2} />
          {orderByDetails && t(orderByDetails.i18n_label)}
        </div>
      }
      placement="bottom-end"
      closeOnSelect
    >
      <OrderByMenuItems options={PROJECT_ORDER_BY_OPTIONS} value={value} onChange={onChange} />
    </CustomMenu>
  );
}
