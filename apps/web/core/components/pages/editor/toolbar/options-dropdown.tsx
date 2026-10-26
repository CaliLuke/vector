/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { useRouter } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";
import { ArrowUpToLine, Clipboard, History } from "lucide-react";
// plane imports
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
// hooks

import { usePageFilters } from "@/hooks/use-page-filters";
import { usePagePermissions } from "@/hooks/use-page-permissions";
import { useQueryParams } from "@/hooks/use-query-params";
import { useWorkspaceMarkdownCopy } from "@/hooks/use-workspace-markdown-copy";
// plane web imports
import type { TPageNavigationPaneTab } from "@/components/pages/navigation-pane/tab-panels";
import { usePageRecord } from "@/components/pages/page-record-context";
// local imports
import { PageActions } from "../../dropdowns";
import { ExportPageModal } from "../../modals/export-page-modal";
import { PAGE_NAVIGATION_PANE_TABS_QUERY_PARAM } from "../../navigation-pane";

type Props = {
  page: TPage;
  workspaceSlug: string;
};

export function PageOptionsDropdown(props: Props) {
  const { page, workspaceSlug } = props;
  const { editorRef } = usePageRecord();
  const { isContentEditable } = usePagePermissions(page, workspaceSlug);
  const {
    actionLabel: markdownActionLabel,
    actionDisabled: markdownActionDisabled,
    handleAction,
  } = useWorkspaceMarkdownCopy(workspaceSlug);
  const getCurrentEditorRef = useCallback(() => editorRef, [editorRef]);
  // states
  const [isExportModalOpen, setIsExportModalOpen] = useState(false);
  // navigation
  const router = useRouter();
  // store values
  const { name } = page;
  // page filters
  const { isFullWidth, handleFullWidth, isStickyToolbarEnabled, handleStickyToolbar } = usePageFilters();
  // query params
  const { updateQueryParams } = useQueryParams();
  // menu items list
  const EXTRA_MENU_OPTIONS = useMemo(
    function EXTRA_MENU_OPTIONS(): React.ComponentProps<typeof PageActions>["extraOptions"] {
      return [
        {
          key: "full-screen",
          action: () => handleFullWidth(!isFullWidth),
          customContent: (
            <>
              Full width
              <span
                aria-hidden="true"
                className={`relative inline-flex h-4 w-7 flex-shrink-0 rounded-full border border-subtle transition-colors duration-200 ease-in-out ${
                  isFullWidth ? "bg-accent-primary" : "bg-(--text-color-icon-placeholder)"
                }`}
              >
                <span
                  className={`inline-block h-3 w-3 transform self-center rounded-full bg-(--text-color-icon-on-color) transition duration-200 ease-in-out ${
                    isFullWidth ? "translate-x-3.5" : "translate-x-0.5"
                  }`}
                />
              </span>
              <span className="sr-only">{isFullWidth ? "On" : "Off"}</span>
            </>
          ),
          className: "flex items-center justify-between gap-2",
        },
        {
          key: "sticky-toolbar",
          action: () => handleStickyToolbar(!isStickyToolbarEnabled),
          customContent: (
            <>
              Sticky toolbar
              <span
                aria-hidden="true"
                className={`relative inline-flex h-4 w-7 flex-shrink-0 rounded-full border border-subtle transition-colors duration-200 ease-in-out ${
                  isStickyToolbarEnabled ? "bg-accent-primary" : "bg-(--text-color-icon-placeholder)"
                }`}
              >
                <span
                  className={`inline-block h-3 w-3 transform self-center rounded-full bg-(--text-color-icon-on-color) transition duration-200 ease-in-out ${
                    isStickyToolbarEnabled ? "translate-x-3.5" : "translate-x-0.5"
                  }`}
                />
              </span>
              <span className="sr-only">{isStickyToolbarEnabled ? "On" : "Off"}</span>
            </>
          ),
          className: "flex items-center justify-between gap-2",
          shouldRender: isContentEditable,
        },
        {
          key: "copy-markdown",
          action: () =>
            handleAction(getCurrentEditorRef, () => {
              setToast({
                type: TOAST_TYPE.SUCCESS,
                title: "Success!",
                message: "Markdown copied to clipboard.",
              });
            }),
          title: markdownActionLabel,
          icon: Clipboard,
          disabled: markdownActionDisabled,
          shouldRender: true,
        },
        {
          key: "version-history",
          action: () => {
            // update query param to show info tab in navigation pane
            const updatedRoute = updateQueryParams({
              paramsToAdd: {
                [PAGE_NAVIGATION_PANE_TABS_QUERY_PARAM]: "info" satisfies TPageNavigationPaneTab,
              },
            });
            void router.navigate({ href: updatedRoute });
          },
          title: "Version history",
          icon: History,
          shouldRender: true,
        },
        {
          key: "export",
          action: () => setIsExportModalOpen(true),
          title: "Export",
          icon: ArrowUpToLine,
          shouldRender: true,
        },
      ];
    },
    [
      handleFullWidth,
      isFullWidth,
      handleStickyToolbar,
      isStickyToolbarEnabled,
      isContentEditable,
      getCurrentEditorRef,
      markdownActionLabel,
      markdownActionDisabled,
      handleAction,
      updateQueryParams,
      router,
      setIsExportModalOpen,
    ]
  );

  return (
    <>
      <ExportPageModal
        editorRef={editorRef}
        isOpen={isExportModalOpen}
        onClose={() => setIsExportModalOpen(false)}
        pageTitle={name ?? ""}
      />
      <PageActions
        workspaceSlug={workspaceSlug}
        extraOptions={EXTRA_MENU_OPTIONS}
        optionsOrder={[
          "full-screen",
          "sticky-toolbar",
          "copy-markdown",
          "version-history",
          "make-a-copy",
          "archive-restore",
          "delete",
          "toggle-access",
          "export",
        ]}
        page={page}
      />
    </>
  );
}
