/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { Link } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { IWebhookRecord } from "@/queries/webhooks";
import { updateWebhookMutationOptions } from "@/queries/webhooks";
import { ToggleSwitch } from "@plane/ui";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";

interface IWebhookListItem {
  webhook: IWebhookRecord;
  workspaceSlug: string;
}

export function WebhooksListItem({ webhook, workspaceSlug }: IWebhookListItem) {
  const queryClient = useQueryClient();
  const updateWebhook = useMutation(updateWebhookMutationOptions(queryClient, workspaceSlug, webhook.id));

  const handleToggle = () => {
    updateWebhook.mutate(
      { is_active: !webhook.is_active },
      {
        onError: (error) =>
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Error!",
            message:
              error && typeof error === "object" && "error" in error && typeof error.error === "string"
                ? error.error
                : "Webhook could not be updated. Please try again.",
          }),
      }
    );
  };

  return (
    <div className="rounded-lg border border-subtle bg-layer-2 px-4 py-3">
      <Link
        to="/$workspaceSlug/settings/webhooks/$webhookId/"
        params={{ workspaceSlug, webhookId: webhook.id }}
        className="flex items-center justify-between gap-4"
      >
        <h5 className="truncate text-body-sm-medium">{webhook.url}</h5>
        <div className="shrink-0">
          <ToggleSwitch value={webhook.is_active} onChange={handleToggle} disabled={updateWebhook.isPending} />
        </div>
      </Link>
    </div>
  );
}
