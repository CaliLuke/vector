/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPageNavigationTabs } from "@plane/types";
import { PageListProvider, usePageList } from "./page-record-context";
// local imports
import { PagesListHeaderRoot } from "./header";
import { PagesListMainContent } from "./pages-list-main-content";

type TPageView = {
  children: React.ReactNode;
  pageType: TPageNavigationTabs;
  projectId: string;
  workspaceSlug: string;
};

export function PagesListView(props: TPageView) {
  const { children, pageType, projectId, workspaceSlug } = props;
  return (
    <PageListProvider workspaceSlug={workspaceSlug} projectId={projectId} pageType={pageType}>
      <PagesListContents pageType={pageType} projectId={projectId} workspaceSlug={workspaceSlug}>
        {children}
      </PagesListContents>
    </PageListProvider>
  );
}

function PagesListContents({ children, pageType, projectId, workspaceSlug }: TPageView) {
  const { pages } = usePageList();
  return (
    <div className="relative flex h-full w-full flex-col overflow-hidden">
      {pages.length > 0 && (
        <PagesListHeaderRoot pageType={pageType} projectId={projectId} workspaceSlug={workspaceSlug} />
      )}
      <PagesListMainContent pageType={pageType}>{children}</PagesListMainContent>
    </div>
  );
}
