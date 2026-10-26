/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";

// plane imports
import { ISSUE_LAYOUT_MAP } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CheckIcon } from "@plane/propel/icons";
import { EIssueLayoutTypes } from "@plane/types";
import { getButtonStyling } from "@plane/propel/button";
import { Dropdown } from "@plane/ui";
import { cn } from "@plane/utils/common";
// components
import { IssueLayoutIcon } from "@/components/issues/issue-layouts/layout-icon";

type TLayoutDropDown = {
  onChange: (value: EIssueLayoutTypes) => void;
  value: EIssueLayoutTypes;
  disabledLayouts?: EIssueLayoutTypes[];
};

function getLayoutDetails(value: string | string[] | undefined) {
  if (typeof value !== "string") return ISSUE_LAYOUT_MAP[EIssueLayoutTypes.LIST];
  return (
    Object.entries(ISSUE_LAYOUT_MAP).find(([key]) => key === value)?.[1] ?? ISSUE_LAYOUT_MAP[EIssueLayoutTypes.LIST]
  );
}

function getLayoutType(value: string): EIssueLayoutTypes | undefined {
  return Object.values(EIssueLayoutTypes).find((layout) => layout.valueOf() === value);
}

export const LayoutDropDown = function LayoutDropDown(props: TLayoutDropDown) {
  const { onChange, value, disabledLayouts = [] } = props;
  // plane i18n
  const { t } = useTranslation();
  // derived values
  const availableLayouts = useMemo(
    () => Object.values(ISSUE_LAYOUT_MAP).filter((layout) => !disabledLayouts.includes(layout.key)),
    [disabledLayouts]
  );

  const options = useMemo(
    () =>
      availableLayouts.map((issueLayout) => ({
        data: issueLayout.key,
        value: issueLayout.key,
      })),
    [availableLayouts]
  );

  const buttonContent = useCallback(
    (_isOpen: boolean, buttonValue: string | string[] | undefined) => {
      const dropdownValue = getLayoutDetails(buttonValue);
      return (
        <div className="flex items-center gap-2 text-secondary">
          <IssueLayoutIcon layout={dropdownValue.key} strokeWidth={2} className={`size-3.5 text-secondary`} />
          <span className="text-11 font-medium">{t(dropdownValue.i18n_label)}</span>
        </div>
      );
    },
    [t]
  );

  const itemContent = useCallback(
    (item: { value: string; selected: boolean }) => {
      const dropdownValue = getLayoutDetails(item.value);

      return (
        <div className={cn("flex w-full items-center justify-between gap-2 text-secondary")}>
          <div className="flex items-center gap-2">
            <IssueLayoutIcon layout={dropdownValue.key} strokeWidth={2} className={`size-3 text-secondary`} />
            <span className="text-11 font-medium">{t(dropdownValue.i18n_label)}</span>
          </div>
          {item.selected && <CheckIcon className="h-3.5 w-3.5 flex-shrink-0" />}
        </div>
      );
    },
    [t]
  );

  const keyExtractor = useCallback((option: { data: unknown; value: string }) => option.value, []);

  return (
    <Dropdown
      onChange={(selected) => {
        const layout = getLayoutType(selected);
        if (layout) onChange(layout);
      }}
      value={value?.toString()}
      keyExtractor={keyExtractor}
      options={options}
      buttonContainerClassName={cn(getButtonStyling("secondary", "lg"))}
      buttonContent={buttonContent}
      renderItem={itemContent}
      disableSearch
    />
  );
};
