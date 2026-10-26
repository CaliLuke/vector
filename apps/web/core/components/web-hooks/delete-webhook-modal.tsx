/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { AlertModalCore } from "@plane/ui";
import { deleteWebhookMutationOptions } from "@/queries/webhooks";

type Props = {
  isOpen: boolean;
  workspaceSlug: string;
  webhookId: string;
  onClose: () => void;
};

export function DeleteWebhookModal({ isOpen, onClose, workspaceSlug, webhookId }: Props) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const deleteWebhook = useMutation(deleteWebhookMutationOptions(queryClient, workspaceSlug, webhookId));
  const activeScope = useRef({ isOpen, workspaceSlug, webhookId });

  useEffect(() => {
    activeScope.current = { isOpen, workspaceSlug, webhookId };
    return () => {
      activeScope.current = { isOpen: false, workspaceSlug, webhookId };
    };
  }, [isOpen, workspaceSlug, webhookId]);

  const isCurrentDialog = () => {
    const current = activeScope.current;
    return current.isOpen && current.workspaceSlug === workspaceSlug && current.webhookId === webhookId;
  };

  const handleDelete = () => {
    deleteWebhook.mutate(undefined, {
      onSuccess: (_result, _variables, context) => {
        if (!context?.sessionCurrent || !isCurrentDialog()) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Webhook deleted successfully.",
        });
        void router.navigate({ href: `/${workspaceSlug}/settings/webhooks/`, replace: true });
      },
      onError: () => {
        if (!isCurrentDialog()) return;
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Webhook could not be deleted. Please try again.",
        });
      },
    });
  };

  return (
    <AlertModalCore
      handleClose={() => {
        if (!deleteWebhook.isPending) onClose();
      }}
      handleSubmit={handleDelete}
      isSubmitting={deleteWebhook.isPending}
      isOpen={isOpen}
      title="Delete webhook"
      content={
        <>
          Are you sure you want to delete this webhook? Future events will not be delivered to this webhook. This action
          cannot be undone.
        </>
      }
    />
  );
}
