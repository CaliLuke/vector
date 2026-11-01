/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useTranslation } from "@plane/i18n";
import { cn } from "@plane/utils/common";

type Props = {
  layout: "list" | "kanban";
  loadedCount: number;
  totalCount: number;
  nextPageResults: boolean | undefined;
  manual: boolean;
  isPaginating: boolean;
  loadMore: () => void;
  loader: ReactNode;
  sentinel: ReactNode;
};

export function GroupPagination({
  layout,
  loadedCount,
  totalCount,
  nextPageResults,
  manual,
  isPaginating,
  loadMore,
  loader,
  sentinel,
}: Props) {
  const { t } = useTranslation();
  const hasMore = nextPageResults === undefined ? loadedCount < totalCount : nextPageResults;
  if (!hasMore) return null;
  if (manual)
    return isPaginating ? (
      loader
    ) : (
      <button
        type="button"
        onClick={loadMore}
        className={cn(
          "w-full cursor-pointer p-3 text-left text-13 font-medium text-accent-primary hover:text-accent-secondary hover:underline",
          layout === "list"
            ? "relative flex h-11 items-center gap-3 border border-transparent border-t-subtle-1 bg-surface-1 pl-8"
            : "sticky bottom-0"
        )}
      >
        {t("common.load_more")} &darr;
      </button>
    );
  const placeholders = (
    <>
      {loader}
      {loader}
      {sentinel}
    </>
  );
  return layout === "list" ? placeholders : <div className="flex flex-col gap-2">{placeholders}</div>;
}
