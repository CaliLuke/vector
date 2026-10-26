/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import type { IWebhookRecord } from "@/queries/webhooks";
import { WebhooksListItem } from "./webhooks-list-item";

type Props = {
  workspaceSlug: string;
  webhooks: IWebhookRecord[];
};

export function WebhooksList({ workspaceSlug, webhooks }: Props) {
  return (
    <div className="flex size-full flex-col gap-y-2 overflow-y-auto rounded-lg border border-subtle bg-layer-1 p-3">
      {webhooks.map((webhook) => (
        <WebhooksListItem key={webhook.id} webhook={webhook} workspaceSlug={workspaceSlug} />
      ))}
    </div>
  );
}
