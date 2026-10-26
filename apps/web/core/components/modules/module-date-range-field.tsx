/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IModule } from "@plane/types";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import type { useModuleActions } from "@/components/modules/use-module-actions";

type Props = {
  moduleDetails: IModule;
  isDisabled: boolean;
  placeholder: { from: string; to: string };
  handleModuleDetailsChange: ReturnType<typeof useModuleActions>["handleModuleDetailsChange"];
  mergeDates?: boolean;
};

export function ModuleDateRangeField(props: Props) {
  const { moduleDetails, isDisabled, placeholder, handleModuleDetailsChange, mergeDates = false } = props;
  const hasDate = Boolean(moduleDetails.start_date) || Boolean(moduleDetails.target_date);

  return (
    <DateRangeDropdown
      buttonContainerClassName={`h-6 w-full flex ${isDisabled ? "cursor-not-allowed" : "cursor-pointer"} items-center gap-1.5 text-tertiary border-[0.5px] border-strong rounded-sm text-11`}
      buttonVariant="transparent-with-text"
      className="h-7"
      value={{
        from: getDate(moduleDetails.start_date),
        to: getDate(moduleDetails.target_date),
      }}
      onSelect={(value) => {
        void handleModuleDetailsChange({
          start_date: value?.from ? renderFormattedPayloadDate(value.from) : null,
          target_date: value?.to ? renderFormattedPayloadDate(value.to) : null,
        });
      }}
      mergeDates={mergeDates}
      placeholder={placeholder}
      disabled={isDisabled}
      hideIcon={{ from: hasDate, to: hasDate }}
    />
  );
}
