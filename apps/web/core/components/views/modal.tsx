/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";

// types
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IProjectView } from "@plane/types";
// ui

// hooks
import { useProjectViewMutations } from "@/hooks/use-project-views";

// local imports
import { ViewForm } from "./form";

type Props = {
  data?: IProjectView | null;
  isOpen: boolean;
  onClose: () => void;
  preLoadedData?: Partial<IProjectView> | null;
  workspaceSlug: string;
  projectId: string;
};

export const CreateUpdateProjectViewModal = function CreateUpdateProjectViewModal(props: Props) {
  const { data, isOpen, onClose, preLoadedData, workspaceSlug, projectId } = props;
  // router
  const router = useRouter();
  const { create, update } = useProjectViewMutations(workspaceSlug, projectId);

  const handleClose = () => {
    onClose();
  };

  const handleCreateView = async (payload: Partial<IProjectView>) => {
    try {
      const res = await create.mutateAsync(payload);
      handleClose();
      void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/views/${res.id}` });
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "View created successfully.",
      });
    } catch (error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to create view. Please try again.",
      });
      throw error;
    }
  };

  const handleUpdateView = async (payload: Partial<IProjectView>) => {
    try {
      await update.mutateAsync({ viewId: data!.id, data: payload });
      handleClose();
    } catch (error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to update view. Please try again.",
      });
      throw error;
    }
  };

  const handleFormSubmit = async (formData: Partial<IProjectView>) => {
    if (!data) await handleCreateView(formData);
    else await handleUpdateView(formData);
  };

  if (!isOpen) return null;
  return (
    <ViewForm
      key={JSON.stringify([workspaceSlug, projectId, data?.id ?? "new"])}
      data={data}
      handleClose={handleClose}
      handleFormSubmit={handleFormSubmit}
      preLoadedData={preLoadedData}
      projectId={projectId}
      workspaceSlug={workspaceSlug}
    />
  );
};
