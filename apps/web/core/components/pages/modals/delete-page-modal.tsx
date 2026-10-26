/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { useParams, useRouter } from "@tanstack/react-router";
// ui

import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { AlertModalCore } from "@plane/ui";
import { getPageName } from "@plane/utils/page";
// constants
// plane web hooks

import { useDeleteProjectPage } from "@/hooks/use-project-pages";
// store

type TConfirmPageDeletionProps = {
  isOpen: boolean;
  onClose: () => void;
  page: TPage;
  projectId?: string;
  workspaceSlug: string;
};

export function DeletePageModal(props: TConfirmPageDeletionProps) {
  const { isOpen, onClose, page, projectId, workspaceSlug } = props;
  // states
  // store hooks
  const remove = useDeleteProjectPage(workspaceSlug, projectId ?? page.project_ids?.[0], page.id);

  // derived values
  const { id: pageId, name } = page;

  const handleClose = () => {
    onClose();
  };

  const router = useRouter();
  const { pageId: routePageId } = useParams({ strict: false });

  const handleDelete = async () => {
    if (!pageId) return;
    try {
      await remove.mutateAsync();
      handleClose();
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Page deleted successfully.",
      });
      if (routePageId) router.history.back();
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Page could not be deleted. Please try again.",
      });
    }
  };

  if (!page || !page.id) return null;

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDelete}
      isSubmitting={remove.isPending}
      isOpen={isOpen}
      title="Delete page"
      content={
        <>
          Are you sure you want to delete page-{" "}
          <span className="font-medium break-words break-all text-primary">{getPageName(name)}</span> ? The Page will be
          deleted permanently. This action cannot be undone.
        </>
      }
    />
  );
}
