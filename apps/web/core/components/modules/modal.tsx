/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IModule } from "@plane/types";
import { ModuleForm } from "./form";
import { useQueryClient } from "@tanstack/react-query";
import { moduleMutationOptions } from "@/queries/cycle-modules";
import { currentParentCatalogueScope, captureParentAccess } from "@/queries/cycle-module-scope";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = { isOpen: boolean; onClose: () => void; data?: IModule; workspaceSlug: string; projectId: string };
export const CreateUpdateModuleModal = function CreateUpdateModuleModal({
  isOpen,
  onClose,
  data,
  workspaceSlug,
  projectId,
}: Props) {
  const { projects } = useWorkspaceProjects(workspaceSlug);
  const workspaceProjectIds = projects?.filter((project) => !project.archived_at).map((project) => project.id);
  const queryClient = useQueryClient(),
    accountId = useCurrentUserId(),
    generation = useIdentityGeneration(queryClient);
  const mounted = useCommittedScope(
    JSON.stringify([accountId, generation, workspaceSlug, projectId, data?.id, isOpen])
  );
  const { isMobile } = usePlatformOS();
  const save = async (payload: Partial<IModule>) => {
    const selectedProject = payload.project_id || projectId;
    const scope = currentParentCatalogueScope(queryClient, workspaceSlug, selectedProject).project;
    const access = captureParentAccess(queryClient, scope);
    try {
      if (!workspaceSlug || !selectedProject) throw new Error("Select a project");
      if (!mounted() || !access.isVerified()) throw new Error("Module editing access changed.");
      const mutation = queryClient.getMutationCache().build(queryClient, moduleMutationOptions(queryClient, scope));
      const result = await mutation.execute(
        data ? { type: "patch", id: data.id, data: payload } : { type: "create", data: payload }
      );
      if (!mounted() || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: data ? "Module updated successfully." : "Module created successfully.",
      });
      onClose();
    } catch (error) {
      if (mounted() && access.isCurrent())
        setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message: "Unable to save the module. Please try again." });
      throw error;
    }
  };
  return isOpen ? (
    <ModuleForm
      key={JSON.stringify([accountId, generation, workspaceSlug, projectId, data?.id])}
      workspaceSlug={workspaceSlug}
      handleFormSubmit={save}
      handleClose={onClose}
      projectId={data?.project_id || projectId || workspaceProjectIds?.[0] || ""}
      data={data}
      isMobile={isMobile}
    />
  ) : null;
};
