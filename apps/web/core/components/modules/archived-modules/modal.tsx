/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useProjectModules } from "@/hooks/use-modules";
import { captureParentAccess } from "@/queries/cycle-module-scope";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveConfirmation } from "@/components/core/archive-confirmation";
type Props = {
  workspaceSlug: string;
  projectId: string;
  moduleId: string;
  handleClose: () => void;
  isOpen: boolean;
  onSubmit?: () => Promise<void>;
};
export function ArchiveModuleModal(props: Props) {
  return props.isOpen ? (
    <ArchiveModuleDialog key={JSON.stringify([props.workspaceSlug, props.projectId, props.moduleId])} {...props} />
  ) : null;
}
function ArchiveModuleDialog({ workspaceSlug, projectId, moduleId, handleClose }: Props) {
  const router = useRouter(),
    client = useQueryClient();
  const { modules, mutation, scope } = useProjectModules(workspaceSlug, projectId);
  const access = captureParentAccess(client, scope);
  return (
    <ArchiveConfirmation
      noun="module"
      name={modules?.find((row) => row.id === moduleId)?.name}
      scopeKey={JSON.stringify([scope, moduleId])}
      isPending={mutation.isPending}
      archive={() => mutation.mutateAsync({ type: "archive", id: moduleId })}
      isCurrent={access.isCurrent}
      onClose={handleClose}
      onArchived={() => {
        handleClose();
        void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/modules` });
      }}
    />
  );
}
