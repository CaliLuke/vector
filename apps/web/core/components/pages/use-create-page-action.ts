/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { EPageAccess } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TPageNavigationTabs } from "@plane/types";
import { useCreateProjectPage } from "@/hooks/use-project-pages";

type Props = {
  onCreated: (pageId: string) => void;
  pageType: TPageNavigationTabs;
  projectId: string;
  workspaceSlug: string;
};

export function useCreatePageAction({ onCreated, pageType, projectId, workspaceSlug }: Props) {
  const create = useCreateProjectPage(workspaceSlug, projectId);
  const mutateAsync = create.mutateAsync;
  const createPage = useCallback(async () => {
    try {
      const page = await mutateAsync({
        access: pageType === "private" ? EPageAccess.PRIVATE : EPageAccess.PUBLIC,
      });
      onCreated(page.id);
    } catch (error) {
      const errorMessage =
        error &&
        typeof error === "object" &&
        "data" in error &&
        error.data &&
        typeof error.data === "object" &&
        "error" in error.data &&
        typeof error.data.error === "string"
          ? error.data.error
          : "Page could not be created. Please try again.";
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: errorMessage,
      });
    }
  }, [mutateAsync, onCreated, pageType]);
  return { createPage, isCreatingPage: create.isPending };
}
