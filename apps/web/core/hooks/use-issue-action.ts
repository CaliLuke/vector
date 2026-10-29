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

type IssueActionScope = {
  workspaceSlug: string | undefined;
  projectId: string | undefined | null;
  issueId: string;
  context: string;
};

type ActionFeedback = {
  success?: string;
  successTitle?: string;
  errorTitle?: string;
  error?: string | ((error: unknown) => string);
};

export function useIssueAction(scope: IssueActionScope, enabled: boolean, messages?: ActionFeedback) {
  const client = useQueryClient();
  const userId = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const { t } = useTranslation();
  const mutationKey = ["issue-property-change", userId, generation, scope] as const;
  const isCurrent = useCommittedScope(JSON.stringify([mutationKey, enabled]));
  const filters = { mutationKey, exact: true };
  const pending = useIsMutating(filters) > 0;
  const { mutateAsync } = useMutation({
    mutationKey,
    mutationFn: (change: () => Promise<unknown>) => change(),
    retry: false,
  });

  const notifyFailure = (
    error: unknown,
    feedback: ActionFeedback | undefined,
    identity: ReturnType<typeof captureIdentityScope>
  ) => {
    if (!isCurrent() || !isIdentityScopeCurrent(client, identity)) return;
    setToast({
      type: TOAST_TYPE.ERROR,
      title: feedback?.errorTitle ?? t("toast.error"),
      message:
        typeof feedback?.error === "function"
          ? feedback.error(error)
          : (feedback?.error ?? t("entity.update.failed", { entity: t("common.work_item") })),
    });
  };

  const execute = (change: () => Promise<unknown>, feedback: ActionFeedback | undefined = messages) => {
    if (!enabled || !isCurrent() || client.isMutating(filters) > 0) return undefined;
    const identity = captureIdentityScope(client, userId);
    return mutateAsync(
      async () => {
        if (!isCurrent() || !isIdentityScopeVerified(client, identity))
          throw new Error("The work item or account session has changed.");
        return change();
      },
      {
        onSuccess: () => {
          if (!feedback?.success || !isCurrent() || !isIdentityScopeCurrent(client, identity)) return;
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: feedback.successTitle ?? t("toast.success"),
            message: feedback.success,
          });
        },
        onError: (error) => notifyFailure(error, feedback, identity),
      }
    );
  };

  const run = (change: () => Promise<unknown>, feedback?: ActionFeedback) => {
    void execute(change, feedback)?.catch(() => undefined);
  };
  const runAsync = async (change: () => Promise<unknown>, feedback: ActionFeedback | undefined = messages) => {
    const completion = execute(change, feedback);
    if (!completion) {
      const error = new Error("The work item action is unavailable or already pending.");
      if (client.isMutating(filters) === 0) notifyFailure(error, feedback, captureIdentityScope(client, userId));
      throw error;
    }
    await completion;
  };
  return { run, runAsync, isCurrent, pending, disabled: !enabled || pending };
}
