/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { useState } from "react";

// ui
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { AlertModalCore } from "@plane/ui";
// hooks
import { useWebhook } from "@/hooks/store/use-webhook";

interface IDeleteWebhook {
  isOpen: boolean;
  onClose: () => void;
}

export function DeleteWebhookModal(props: IDeleteWebhook) {
  const { isOpen, onClose } = props;
  // states
  const [isDeleting, setIsDeleting] = useState(false);
  // router
  const router = useRouter();
  // store hooks
  const { removeWebhook } = useWebhook();

  const { workspaceSlug, webhookId } = useParams({ strict: false });

  const handleClose = () => {
    onClose();
  };

  const handleDelete = async () => {
    if (!workspaceSlug || !webhookId) return;
    setIsDeleting(true);
    try {
      await removeWebhook(workspaceSlug.toString(), webhookId.toString());
      void router.navigate({ href: `/${workspaceSlug}/settings/webhooks/`, replace: true });
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Webhook deleted successfully.",
      });
    } catch (_error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Webhook could not be deleted. Please try again.",
      });
    }
    setIsDeleting(false);
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDelete}
      isSubmitting={isDeleting}
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
