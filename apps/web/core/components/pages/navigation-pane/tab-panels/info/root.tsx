/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { useParams } from "@tanstack/react-router";
// components
// store
// local imports
import { PageNavigationPaneInfoTabActorsInfo } from "./actors-info";
import { PageNavigationPaneInfoTabDocumentInfo } from "./document-info";
import { PageNavigationPaneInfoTabVersionHistory } from "./version-history";

type Props = {
  page: TPage;
};

export function PageNavigationPaneInfoTabPanel(props: Props) {
  const { page } = props;
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  return (
    <div className="flex h-full flex-col px-4">
      <div className="mt-5 flex-1 overflow-y-auto">
        <PageNavigationPaneInfoTabDocumentInfo />
        <PageNavigationPaneInfoTabActorsInfo page={page} workspaceSlug={workspaceSlug} />
        <div className="my-3 h-px flex-shrink-0 bg-layer-1" />
        <PageNavigationPaneInfoTabVersionHistory
          page={page}

          workspaceSlug={workspaceSlug}
        />
      </div>
    </div>
  );
}
