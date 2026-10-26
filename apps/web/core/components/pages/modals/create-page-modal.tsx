/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
// constants
import type { EPageAccess } from "@plane/constants";
import type { TPage } from "@plane/types";
// ui
// hooks

import { useCreateProjectPage } from "@/hooks/use-project-pages";
// local imports
import { PageForm } from "./page-form";

type Props = {
  workspaceSlug: string;
  projectId: string;
  isModalOpen: boolean;
  pageAccess?: EPageAccess;
  handleModalClose: () => void;
  redirectionEnabled?: boolean;
};

export function CreatePageModal(props: Props) {
  const { workspaceSlug, projectId, isModalOpen, pageAccess, handleModalClose, redirectionEnabled = false } = props;
  // router
  const router = useRouter();
  // store hooks
  const create = useCreateProjectPage(workspaceSlug, projectId);
  const handleFormSubmit = async (values: Partial<TPage>) => {
    const page = await create.mutateAsync(values);
    handleModalClose();
    if (redirectionEnabled) void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/pages/${page.id}` });
  };
  if (!isModalOpen) return null;

  return (
    <PageForm
      key={`${workspaceSlug}/${projectId}`}
      pageAccess={pageAccess}
      handleModalClose={handleModalClose}
      handleFormSubmit={handleFormSubmit}
    />
  );
}
