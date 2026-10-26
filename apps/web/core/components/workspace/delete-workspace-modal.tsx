/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useRouter } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspace } from "@plane/types";
import { ConfirmActionModal } from "@/components/common/confirm-action-modal";
import { useDeleteWorkspace, useWorkspaces } from "@/hooks/use-workspaces";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useCurrentUserSettings } from "@/hooks/use-current-user-settings";
import { refreshCurrentUserSettings } from "@/queries/current-user-settings";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { getWorkspaceDestination, workspaceKeys } from "@/queries/workspaces";

type Props = { isOpen: boolean; data: IWorkspace | null; onClose: () => void };
export function DeleteWorkspaceModal({ isOpen, data, onClose }: Props) {
  const router = useRouter();
  const routeHref = useLocation({ select: (location) => location.href });
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const { data: currentSettings } = useCurrentUserSettings();
  const { data: workspaces = [] } = useWorkspaces();
  const deletion = useDeleteWorkspace(data?.slug);
  if (!data) return null;
  const identityScope = captureIdentityScope(queryClient, currentUser?.id);
  const canContinue = () =>
    isIdentityScopeCurrent(queryClient, identityScope) && router.state.location.href === routeHref;
  const afterDeletion = async () => {
    if (!canContinue()) return;
    let settings = currentSettings?.id === currentUser?.id ? currentSettings : undefined;
    let remainingWorkspaces = workspaces.filter((workspace) => workspace.id !== data.id);
    setToast({
      type: TOAST_TYPE.SUCCESS,
      title: t("workspace_settings.settings.general.delete_modal.success_title"),
      message: t("workspace_settings.settings.general.delete_modal.success_message"),
    });
    try {
      if (!currentUser?.id) throw new Error("A signed-in user is required to refresh account settings.");
      settings = await refreshCurrentUserSettings(queryClient, currentUser.id);
    } catch {
      if (canContinue()) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Workspace deleted",
          message: "Unable to refresh your account settings. Please reload the page.",
        });
      }
    }
    if (!canContinue()) return;
    if (currentUser?.id) {
      remainingWorkspaces =
        queryClient.getQueryData<IWorkspace[]>(workspaceKeys.list(currentUser.id)) ?? remainingWorkspaces;
    }
    try {
      await router.navigate({ href: getWorkspaceDestination(remainingWorkspaces, settings) });
    } catch {
      if (canContinue()) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Workspace deleted",
          message: "Unable to choose a destination. Please reload the page.",
        });
      }
    }
  };
  return (
    <ConfirmActionModal
      key={data.id}
      isOpen={isOpen}
      title={t("workspace_settings.settings.general.delete_modal.title")}
      name={data.name}
      confirmation="delete my workspace"
      actionLabel={t("confirm")}
      pendingLabel={t("deleting")}
      errorMessage={t("workspace_settings.settings.general.delete_modal.error_message")}
      description={
        <>
          You are about to delete <strong>{data.name}</strong>. You will lose access to all your work data in this
          workspace without any way to restore it.
        </>
      }
      onClose={onClose}
      onConfirm={async () => {
        await deletion.mutateAsync();
        await afterDeletion();
      }}
    />
  );
}
