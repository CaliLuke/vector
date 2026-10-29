/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIsMutating, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { captureIdentityScope, isIdentityScopeCurrent, isIdentityScopeVerified } from "@/queries/identity";

type PropertyScope = {
  workspaceSlug: string | undefined;
  projectId: string | undefined | null;
  issueId: string;
  context: string;
};

type PropertyFeedback = { success?: string; error?: string };

export function useIssuePropertyChange(scope: PropertyScope, enabled: boolean, messages?: PropertyFeedback) {
  const client = useQueryClient();
  const userId = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const { t } = useTranslation();
  const mutationKey = ["issue-property-change", userId, generation, scope] as const;
  const isCurrent = useCommittedScope(JSON.stringify([mutationKey, enabled]));
  const filters = { mutationKey, exact: true };
  const pending = useIsMutating(filters) > 0;
  const { mutate } = useMutation({
    mutationKey,
    mutationFn: (change: () => Promise<unknown>) => change(),
    retry: false,
  });

  const run = (change: () => Promise<unknown>, feedback: PropertyFeedback | undefined = messages) => {
    if (!enabled || !isCurrent() || client.isMutating(filters) > 0) return;
    const identity = captureIdentityScope(client, userId);
    mutate(
      async () => {
        if (!isCurrent() || !isIdentityScopeVerified(client, identity))
          throw new Error("The work item or account session has changed.");
        return change();
      },
      {
        onSuccess: () => {
          if (!feedback?.success || !isCurrent() || !isIdentityScopeCurrent(client, identity)) return;
          setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: feedback.success });
        },
        onError: () => {
          if (!isCurrent() || !isIdentityScopeCurrent(client, identity)) return;
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: feedback?.error ?? t("entity.update.failed", { entity: t("common.work_item") }),
          });
        },
      }
    );
  };

  return { run, pending, disabled: !enabled || pending };
}
