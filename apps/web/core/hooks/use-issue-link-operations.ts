/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import type { TIssueLink, TIssueServiceType } from "@plane/types";
import { useIssueLinks } from "@/hooks/use-issue-resources";
import { useIssueAction } from "@/hooks/use-issue-action";

export type TLinkOperations = {
  create: (data: Partial<TIssueLink>) => Promise<void>;
  update: (linkId: string, data: Partial<TIssueLink>) => Promise<void>;
  remove: (linkId: string) => Promise<void>;
  pending: boolean;
  isCurrent: () => boolean;
};

export function useLinkOperations(
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  issueServiceType: TIssueServiceType
): TLinkOperations {
  const { t } = useTranslation();
  const links = useIssueLinks({ workspaceSlug, projectId, issueId, serviceType: issueServiceType });
  const action = useIssueAction(
    { workspaceSlug, projectId, issueId, context: `links:${issueServiceType}` },
    links.isReady
  );
  return {
    create: (data) =>
      action.runAsync(() => links.create(data), {
        successTitle: t("links.toasts.created.title"),
        success: t("links.toasts.created.message"),
        errorTitle: t("links.toasts.not_created.title"),
        error: (error) => getLinkErrorMessage(error) ?? t("links.toasts.not_created.message"),
      }),
    update: (linkId, data) =>
      action.runAsync(() => links.update(linkId, data), {
        successTitle: t("links.toasts.updated.title"),
        success: t("links.toasts.updated.message"),
        errorTitle: t("links.toasts.not_updated.title"),
        error: (error) => getLinkErrorMessage(error) ?? t("links.toasts.not_updated.message"),
      }),
    remove: (linkId) =>
      action.runAsync(() => links.remove(linkId), {
        successTitle: t("links.toasts.removed.title"),
        success: t("links.toasts.removed.message"),
        errorTitle: t("links.toasts.not_removed.title"),
        error: t("links.toasts.not_removed.message"),
      }),
    pending: action.pending,
    isCurrent: action.isCurrent,
  };
}

function getLinkErrorMessage(error: unknown): string | undefined {
  if (!error || typeof error !== "object" || !("data" in error)) return undefined;
  const data = error.data;
  if (data && typeof data === "object" && "error" in data && typeof data.error === "string") return data.error;
  return undefined;
}
