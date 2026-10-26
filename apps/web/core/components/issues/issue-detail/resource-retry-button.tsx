/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";

export function IssueResourceRetryButton({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation();
  return (
    <button type="button" className="text-11 text-danger-primary" onClick={onRetry}>
      {t("common.retry")}
    </button>
  );
}
