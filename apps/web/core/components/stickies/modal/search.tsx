/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// helpers
import { useTranslation } from "@plane/i18n";
import { useStickies } from "@/components/stickies/provider";
import { ControlledExpandableSearch } from "@/components/common/controlled-expandable-search";

export const StickySearch = function StickySearch() {
  // router
  // hooks
  const { searchQuery, setSearchQuery } = useStickies();
  const { t } = useTranslation();
  return (
    <ControlledExpandableSearch
      value={searchQuery}
      onChange={setSearchQuery}
      placeholder={t("stickies.search_placeholder")}
      className="my-auto mr-2 flex items-center"
      triggerClassName="-mr-2"
      fieldClassName="gap-1"
      expandedFieldClassName="w-30 border-subtle px-2.5 py-1.5 md:w-64"
      searchIconClassName="size-3.5 shrink-0"
    />
  );
};
