/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

// local imports
import { PageOptionsDropdown } from "../editor/toolbar";
import { PageArchivedBadge } from "./archived-badge";
import { PageCopyLinkControl } from "./copy-link-control";
import { PageFavoriteControl } from "./favorite-control";
import { PageOfflineBadge } from "./offline-badge";
import { PageLockControl } from "./lock-control";

type Props = {
  page: TPage;
  workspaceSlug: string;
};

export function PageHeaderActions(props: Props) {
  const { page, workspaceSlug } = props;

  return (
    <div className="flex items-center gap-1">
      <PageArchivedBadge page={page} />
      <PageOfflineBadge page={page} workspaceSlug={workspaceSlug} />
      <PageLockControl page={page} workspaceSlug={workspaceSlug} />
      <PageCopyLinkControl page={page} workspaceSlug={workspaceSlug} />
      <PageFavoriteControl page={page} workspaceSlug={workspaceSlug} />
      <PageOptionsDropdown page={page} workspaceSlug={workspaceSlug} />
    </div>
  );
}
