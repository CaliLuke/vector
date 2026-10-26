/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useProjectCycles } from "@/hooks/use-cycles";
import { captureParentAccess } from "@/queries/cycle-module-scope";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveConfirmation } from "@/components/core/archive-confirmation";
type Props = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string;
  handleClose: () => void;
  isOpen: boolean;
  onSubmit?: () => Promise<void>;
};
export function ArchiveCycleModal(props: Props) {
  return props.isOpen ? (
    <ArchiveCycleDialog key={JSON.stringify([props.workspaceSlug, props.projectId, props.cycleId])} {...props} />
  ) : null;
}
function ArchiveCycleDialog({ workspaceSlug, projectId, cycleId, handleClose }: Props) {
  const router = useRouter(),
    client = useQueryClient();
  const { cycles, mutation, scope } = useProjectCycles(workspaceSlug, projectId);
  const access = captureParentAccess(client, scope);
  return (
    <ArchiveConfirmation
      noun="cycle"
      name={cycles?.find((row) => row.id === cycleId)?.name}
      scopeKey={JSON.stringify([scope, cycleId])}
      isPending={mutation.isPending}
      archive={() => mutation.mutateAsync({ type: "archive", id: cycleId })}
      isCurrent={access.isCurrent}
      onClose={handleClose}
      onArchived={() => {
        handleClose();
        void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/cycles` });
      }}
    />
  );
}
