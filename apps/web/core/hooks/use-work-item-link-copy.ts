/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { copyUrlToClipboard } from "@plane/utils/string";
import { useIssueAction } from "@/hooks/use-issue-action";

export function useWorkItemLinkCopy(scope: Omit<Parameters<typeof useIssueAction>[0], "context">, path: string) {
  const { t } = useTranslation();
  const action = useIssueAction(
    { ...scope, context: `copy-link:${path}` },
    Boolean(scope.workspaceSlug && scope.projectId && scope.issueId && path),
    {
      successTitle: t("common.link_copied"),
      success: t("common.copied_to_clipboard"),
      error: t("power_k.contextual_actions.work_item.copy_url_toast_error"),
    }
  );
  return { copy: () => action.run(() => copyUrlToClipboard(path)), disabled: action.disabled };
}
