/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

// components
// store
// local imports
import { PageNavigationPaneAssetsTabPanel } from "./assets";
import { PageNavigationPaneInfoTabPanel } from "./info/root";
import { PageNavigationPaneOutlineTabPanel } from "./outline";
import { Tabs } from "@plane/propel/tabs";
import { ORDERED_PAGE_NAVIGATION_TABS_LIST } from ".";

type Props = {
  page: TPage;
};

export function PageNavigationPaneTabPanelsRoot(props: Props) {
  const { page } = props;

  return (
    <>
      {ORDERED_PAGE_NAVIGATION_TABS_LIST.map((tab) => (
        <Tabs.Content key={tab.key} value={tab.key} className="flex-1 overflow-hidden py-2">
          {tab.key === "outline" && <PageNavigationPaneOutlineTabPanel />}
          {tab.key === "info" && <PageNavigationPaneInfoTabPanel page={page} />}
          {tab.key === "assets" && <PageNavigationPaneAssetsTabPanel page={page} />}
        </Tabs.Content>
      ))}
    </>
  );
}
