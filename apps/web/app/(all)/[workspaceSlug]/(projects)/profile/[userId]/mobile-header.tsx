/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane constants
import { ISSUE_LAYOUTS } from "@plane/constants";
// plane i18n
import { useTranslation } from "@plane/i18n";
// icons
import { ChevronDownIcon } from "@plane/propel/icons";
// ui
import { CustomMenu } from "@plane/ui";
// components
import { DisplayFiltersSelection, FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { IssueLayoutIcon } from "@/components/issues/issue-layouts/layout-icon";
// hooks
import { useProfileIssueDisplayPreferences } from "@/hooks/use-profile-issue-display-preferences";

export function ProfileIssuesMobileHeader() {
  // plane i18n
  const { t } = useTranslation();
  const {
    handleDisplayFilters,
    handleDisplayProperties,
    handleLayoutChange,
    issueFilters,
    layoutDisplayFiltersOptions,
  } = useProfileIssueDisplayPreferences();

  return (
    <div className="flex justify-evenly border-b border-subtle py-2 md:hidden">
      <CustomMenu
        maxHeight={"md"}
        className="flex flex-grow justify-center text-13 text-secondary"
        placement="bottom-start"
        customButton={
          <div className="flex-center flex text-13 text-secondary">
            {t("common.layout")}
            <ChevronDownIcon className="my-auto ml-2 h-4 w-4 text-secondary" strokeWidth={2} />
          </div>
        }
        customButtonClassName="flex flex-center text-secondary text-13"
        closeOnSelect
      >
        {ISSUE_LAYOUTS.map((layout) => {
          if (PROFILE_MOBILE_EXCLUDED_LAYOUTS.has(layout.key)) return null;
          return (
            <CustomMenu.MenuItem
              key={layout.key}
              onClick={() => handleLayoutChange(layout.key)}
              className="flex items-center gap-2"
            >
              <IssueLayoutIcon layout={layout.key} className="h-3 w-3" />
              <div className="text-tertiary">{t(layout.i18n_title)}</div>
            </CustomMenu.MenuItem>
          );
        })}
      </CustomMenu>
      <div className="flex flex-grow items-center justify-center border-l border-subtle text-13 text-secondary">
        <FiltersDropdown
          title={t("common.display")}
          placement="bottom-end"
          menuButton={
            <div className="flex-center flex text-13 text-secondary">
              {t("common.display")}
              <ChevronDownIcon className="ml-2 h-4 w-4 text-secondary" strokeWidth={2} />
            </div>
          }
        >
          <DisplayFiltersSelection
            layoutDisplayFiltersOptions={layoutDisplayFiltersOptions}
            displayFilters={issueFilters?.displayFilters ?? {}}
            handleDisplayFiltersUpdate={handleDisplayFilters}
            displayProperties={issueFilters?.displayProperties ?? {}}
            handleDisplayPropertiesUpdate={handleDisplayProperties}
          />
        </FiltersDropdown>
      </div>
    </div>
  );
}

const PROFILE_MOBILE_EXCLUDED_LAYOUTS = new Set<string>(["spreadsheet", "gantt_chart", "calendar"]);
