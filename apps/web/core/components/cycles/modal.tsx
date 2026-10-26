/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ICycle } from "@plane/types";
import { cycleMutationOptions } from "@/queries/cycle-modules";
import { currentParentCatalogueScope, captureParentAccess } from "@/queries/cycle-module-scope";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { CycleForm } from "./form";

type CycleModalProps = {
  isOpen: boolean;
  handleClose: () => void;
  data?: ICycle | null;
  workspaceSlug: string;
  projectId: string;
};

export function CycleCreateUpdateModal({ isOpen, handleClose, data, workspaceSlug, projectId }: CycleModalProps) {
  const queryClient = useQueryClient();
  const { isMobile } = usePlatformOS();
  const accountId = useCurrentUserId(),
    generation = useIdentityGeneration(queryClient);
  const mounted = useCommittedScope(
    JSON.stringify([accountId, generation, workspaceSlug, projectId, data?.id, isOpen])
  );
  const saveCycle = async (payload: Partial<ICycle>) => {
    const selectedProject = payload.project_id ?? projectId;
    const scope = currentParentCatalogueScope(queryClient, workspaceSlug, selectedProject).project;
    const access = captureParentAccess(queryClient, scope);
    if (!mounted() || !access.isVerified()) throw new Error("Cycle editing access changed.");
    const mutation = queryClient.getMutationCache().build(queryClient, cycleMutationOptions(queryClient, scope));
    const result = await mutation.execute(
      data ? { type: "patch", id: data.id, data: payload } : { type: "create", data: payload }
    );
    if (!mounted() || !result.captured.isCurrent()) return;
    setToast({
      type: TOAST_TYPE.SUCCESS,
      title: "Success!",
      message: data ? "Cycle updated successfully." : "Cycle created successfully.",
    });
    handleClose();
  };
  return isOpen ? (
    <CycleForm
      key={JSON.stringify([accountId, generation, workspaceSlug, projectId, data?.id])}
      handleFormSubmit={saveCycle}
      handleClose={handleClose}
      projectId={projectId}
      data={data}
      isMobile={isMobile}
    />
  ) : null;
}
