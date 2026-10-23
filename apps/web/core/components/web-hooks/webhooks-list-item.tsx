/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Vector imports
import { Link, useParams } from "@tanstack/react-router";
import type { IWebhook } from "@plane/types";
import { ToggleSwitch } from "@plane/ui";
// hooks
import { useWebhook } from "@/hooks/store/use-webhook";

interface IWebhookListItem {
  webhook: IWebhook;
}

export function WebhooksListItem(props: IWebhookListItem) {
  const { webhook } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const { updateWebhook } = useWebhook();

  const handleToggle = async () => {
    if (!workspaceSlug || !webhook.id) return;
    await updateWebhook(workspaceSlug.toString(), webhook.id, { is_active: !webhook.is_active });
  };

  return (
    <div className="rounded-lg border border-subtle bg-layer-2 px-4 py-3">
      <Link
        to="/$workspaceSlug/settings/webhooks/$webhookId/"
        params={{ workspaceSlug: workspaceSlug, webhookId: webhook?.id }}
        className="flex items-center justify-between gap-4"
      >
        <h5 className="truncate text-body-sm-medium">{webhook.url}</h5>
        <div className="shrink-0">
          <ToggleSwitch value={webhook.is_active} onChange={handleToggle} />
        </div>
      </Link>
    </div>
  );
}
