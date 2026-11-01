/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

//ui
import { ArrowDownWideNarrow, ArrowUpNarrowWide, CheckIcon, ChevronDownIcon, Eraser, MoveRight } from "lucide-react";
// constants
import { SPREADSHEET_PROPERTY_DETAILS } from "@plane/constants";
// i18n
import { useTranslation } from "@plane/i18n";
// types
import type { TIssueStandardDisplayProperty, IIssueDisplayFilterOptions, TIssueOrderByOptions } from "@plane/types";
import { CustomMenu, Row } from "@plane/ui";
import { useLocalStorage } from "@plane/hooks";
import { SpreadSheetPropertyIcon } from "../../utils";

interface Props {
  property: TIssueStandardDisplayProperty;
  displayFilters: IIssueDisplayFilterOptions;
  handleDisplayFilterUpdate: (data: Partial<IIssueDisplayFilterOptions>) => void;
  onClose: () => void;
  isEpic?: boolean;
}

export function HeaderColumn(props: Props) {
  const { displayFilters, handleDisplayFilterUpdate, property, onClose, isEpic = false } = props;
  // i18n
  const { t } = useTranslation();
  const { storedValue: selectedMenuItem, setValue: setSelectedMenuItem } = useLocalStorage(
    "spreadsheetViewSorting",
    ""
  );
  const { storedValue: activeSortingProperty, setValue: setActiveSortingProperty } = useLocalStorage(
    "spreadsheetViewActiveSortingProperty",
    ""
  );
  const propertyDetails = SPREADSHEET_PROPERTY_DETAILS[property];

  const handleOrderBy = (order: TIssueOrderByOptions, itemKey: string) => {
    handleDisplayFilterUpdate({ order_by: order });

    setSelectedMenuItem(`${order}_${itemKey}`);
    setActiveSortingProperty(order === "-created_at" ? "" : itemKey);
  };

  if (!propertyDetails) return null;

  return (
    <CustomMenu
      customButtonClassName="clickable !w-full"
      customButtonTabIndex={-1}
      className="!w-full"
      customButton={
        <Row className="flex w-full cursor-pointer items-center justify-between gap-1.5 py-2 text-13 text-secondary hover:text-primary">
          <div className="flex items-center gap-1.5">
            {<SpreadSheetPropertyIcon iconKey={propertyDetails.icon} className="h-4 w-4 text-placeholder" />}
            {property === "sub_issue_count" && isEpic ? t("issue.label", { count: 2 }) : t(propertyDetails.i18n_title)}
          </div>
          <div className="ml-3 flex">
            {activeSortingProperty === property && (
              <div className="flex h-3.5 w-3.5 items-center justify-center rounded-full">
                {propertyDetails.ascendingOrderKey === displayFilters.order_by ? (
                  <ArrowDownWideNarrow className="h-3 w-3" />
                ) : (
                  <ArrowUpNarrowWide className="h-3 w-3" />
                )}
              </div>
            )}
            <ChevronDownIcon className="h-3 w-3" aria-hidden="true" />
          </div>
        </Row>
      }
      onMenuClose={onClose}
      placement="bottom-start"
      closeOnSelect
    >
      {[
        {
          order: propertyDetails.ascendingOrderKey,
          from: propertyDetails.ascendingOrderTitle,
          to: propertyDetails.descendingOrderTitle,
          Icon: ArrowDownWideNarrow,
        },
        {
          order: propertyDetails.descendingOrderKey,
          from: propertyDetails.descendingOrderTitle,
          to: propertyDetails.ascendingOrderTitle,
          Icon: ArrowUpNarrowWide,
        },
      ].map(({ order, from, to, Icon }) => {
        const selected = selectedMenuItem === `${order}_${property}`;
        return (
          <CustomMenu.MenuItem key={order} onClick={() => handleOrderBy(order, property)}>
            <div
              className={`flex items-center justify-between gap-1.5 px-1 ${selected ? "text-primary" : "text-secondary hover:text-primary"}`}
            >
              <div className="flex items-center gap-2">
                <Icon className="h-3 w-3 stroke-[1.5]" />
                <span>{from}</span>
                <MoveRight className="h-3 w-3" />
                <span>{to}</span>
              </div>
              {selected && <CheckIcon className="h-3 w-3" />}
            </div>
          </CustomMenu.MenuItem>
        );
      })}
      {selectedMenuItem && displayFilters.order_by !== "-created_at" && selectedMenuItem.includes(property) && (
        <CustomMenu.MenuItem
          className={`mt-0.5 ${selectedMenuItem === `-created_at_${property}` ? "bg-layer-1" : ""}`}
          key={property}
          onClick={() => handleOrderBy("-created_at", property)}
        >
          <div className="flex items-center gap-2 px-1">
            <Eraser className="h-3 w-3" />
            <span>{t("common.actions.clear_sorting")}</span>
          </div>
        </CustomMenu.MenuItem>
      )}
    </CustomMenu>
  );
}
