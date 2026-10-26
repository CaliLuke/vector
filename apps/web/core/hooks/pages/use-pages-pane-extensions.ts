/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import type { RefObject } from "react";

import type { TPage } from "@plane/types";
import type { EditorRefApi } from "@plane/editor";
import {
  PAGE_NAVIGATION_PANE_TAB_KEYS,
  PAGE_NAVIGATION_PANE_TABS_QUERY_PARAM,
  PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM,
} from "@/components/pages/navigation-pane";

import { useQueryParams } from "@/hooks/use-query-params";
import type { TPageNavigationPaneTab } from "@/components/pages/navigation-pane/tab-panels";
import type { INavigationPaneExtension } from "@/components/pages/navigation-pane";
import { useLocation, useRouter } from "@tanstack/react-router";

export type TPageExtensionHookParams = {
  page: TPage;
  editorRef: RefObject<EditorRefApi | null>;
};

export const usePagesPaneExtensions = (_params: TPageExtensionHookParams) => {
  const router = useRouter();
  const { updateQueryParams } = useQueryParams();
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));

  // Generic navigation pane logic - hook manages feature-specific routing
  const navigationPaneQueryParam = searchParams.get(
    PAGE_NAVIGATION_PANE_TABS_QUERY_PARAM
  ) as TPageNavigationPaneTab | null;

  const isNavigationPaneOpen =
    !!navigationPaneQueryParam && PAGE_NAVIGATION_PANE_TAB_KEYS.includes(navigationPaneQueryParam);

  const handleOpenNavigationPane = useCallback(() => {
    const updatedRoute = updateQueryParams({
      paramsToAdd: { [PAGE_NAVIGATION_PANE_TABS_QUERY_PARAM]: "outline" },
    });
    void router.navigate({ href: updatedRoute });
  }, [router, updateQueryParams]);

  const editorExtensionHandlers: Map<string, unknown> = useMemo(() => {
    const map: Map<string, unknown> = new Map();
    return map;
  }, []);

  const navigationPaneExtensions: INavigationPaneExtension[] = [];

  const handleCloseNavigationPane = useCallback(() => {
    const updatedRoute = updateQueryParams({
      paramsToRemove: [PAGE_NAVIGATION_PANE_TABS_QUERY_PARAM, PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM],
    });
    void router.navigate({ href: updatedRoute });
  }, [router, updateQueryParams]);

  return {
    editorExtensionHandlers,
    navigationPaneExtensions,
    handleOpenNavigationPane,
    isNavigationPaneOpen,
    handleCloseNavigationPane,
  };
};
