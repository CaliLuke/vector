/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { Star } from "lucide-react";
// ui
import { IconButton } from "@plane/propel/icon-button";
// hooks
import { usePageOperations } from "@/hooks/use-page-operations";
import { usePagePermissions } from "@/hooks/use-page-permissions";
// store

type Props = {
  page: TPage;
  workspaceSlug: string;
};

export function PageFavoriteControl({ page, workspaceSlug }: Props) {
  const { canCurrentUserFavoritePage } = usePagePermissions(page, workspaceSlug);
  // page operations
  const { pageOperations, isFavorite, isFavoriteReady } = usePageOperations({
    page,
    workspaceSlug,
  });

  if (!canCurrentUserFavoritePage || !isFavoriteReady) return null;

  return (
    <IconButton
      variant="ghost"
      size="lg"
      icon={Star}
      onClick={() => {
        void pageOperations.toggleFavorite();
      }}
      aria-label={isFavorite ? "Remove favorite" : "Add to favorites"}
      className={
        isFavorite ? "[&_svg]:fill-(--color-label-yellow-icon) [&_svg]:stroke-(--color-label-yellow-icon)" : ""
      }
    />
  );
}
