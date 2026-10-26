/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { Input } from "@plane/ui";

type Props = {
  value: string;
  onChange: (value: string) => void;
  hasError: boolean;
  disabled?: boolean;
};
export function WebhookInput(props: Props) {
  const { value, onChange, hasError, disabled } = props;
  const { t } = useTranslation();

  return (
    <>
      <label htmlFor="webhook-url" className="text-13 font-medium">
        {t("workspace_settings.settings.webhooks.modal.payload")}
      </label>
      <Input
        id="webhook-url"
        disabled={disabled}
        type="url"
        className="h-11 w-full"
        onChange={(e) => onChange(e.target.value)}
        value={value}
        aria-invalid={hasError}
        autoComplete="off"
        hasError={hasError}
        placeholder="https://example.com/post"
      />
    </>
  );
}
