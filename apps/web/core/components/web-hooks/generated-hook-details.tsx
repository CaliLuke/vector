/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import type { IWebhookRecord } from "@/queries/webhooks";
import { WebhookSecretKey } from "./form";

type Props = {
  handleClose: () => void;
  webhookDetails: IWebhookRecord;
  secretKey: string | null;
};

export function GeneratedHookDetails({ handleClose, webhookDetails, secretKey }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="space-y-5 p-5">
        <div className="space-y-3">
          <h3 className="text-18 font-medium text-secondary">{t("workspace_settings.key_created")}</h3>
          <p className="text-13 text-placeholder">{t("workspace_settings.copy_key")}</p>
        </div>
        <WebhookSecretKey data={webhookDetails} secretKey={secretKey} />
      </div>
      <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <Button variant="secondary" size="lg" onClick={handleClose}>
          Close
        </Button>
      </div>
    </>
  );
}
