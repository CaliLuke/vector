/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { useTranslation } from "@plane/i18n";
import { PlusIcon } from "@plane/propel/icons";
// hooks
import { useStickies, StickiesProvider } from "./provider";
// local imports
import { StickiesTruncated } from "./layout/stickies-truncated";
import { StickySearch } from "./modal/search";
import { useStickyOperations } from "./sticky/use-operations";

const StickiesWidgetContent = function StickiesWidgetContent() {
  // params
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const { creatingSticky } = useStickies();
  const { t } = useTranslation();
  // sticky operations
  const { stickyOperations } = useStickyOperations({
    workspaceSlug: workspaceSlug?.toString() ?? "",
  });

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div className="text-14 font-semibold text-tertiary">{t("stickies.title")}</div>
        {/* actions */}
        <div className="flex gap-2">
          <StickySearch />
          <button
            onClick={() => {
              stickyOperations.create();
            }}
            className="my-auto flex gap-1 text-13 font-medium text-accent-primary"
            disabled={creatingSticky}
          >
            <PlusIcon className="my-auto size-4" />
            <span>{t("stickies.add")}</span>
            {creatingSticky && (
              <div
                className="size-4 animate-spin rounded-full border-2 border-accent-strong border-t-transparent"
                role="status"
                aria-label="loading"
              />
            )}
          </button>
        </div>
      </div>
      <div className="-mx-2">
        <StickiesTruncated />
      </div>
    </div>
  );
};

export function StickiesWidget({ workspaceSlug }: { workspaceSlug: string }) {
  return (
    <StickiesProvider key={workspaceSlug} workspaceSlug={workspaceSlug}>
      <StickiesWidgetContent />
    </StickiesProvider>
  );
}
