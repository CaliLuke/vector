/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import type { TPageNavigationTabs } from "@plane/types";
// components
import { ListLayout } from "@/components/core/list";
// plane web hooks
import { usePageList } from "../page-record-context";
// local imports
import { PageListBlock } from "./block";

type TPagesListRoot = {
  pageType: TPageNavigationTabs;
  workspaceSlug: string;
};

export function PagesListRoot(props: TPagesListRoot) {
  const { projectId, visiblePages } = usePageList();
  const { workspaceSlug } = props;
  return (
    <ListLayout>
      {visiblePages.map((page) => (
        <PageListBlock key={page.id} page={page} projectId={projectId} workspaceSlug={workspaceSlug} />
      ))}
    </ListLayout>
  );
}
