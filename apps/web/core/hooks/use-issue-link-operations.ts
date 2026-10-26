/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TIssueLink, TIssueServiceType } from "@plane/types";
// hooks
import { useIssueLinks } from "@/hooks/use-issue-resources";
// local imports
import type { TLinkOperations } from "@/components/issues/issue-detail/links";

export const useLinkOperations = (
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  issueServiceType: TIssueServiceType,
  onSuccess?: () => void
): TLinkOperations => {
  const {
    create: createLink,
    update: updateLink,
    remove: removeLink,
  } = useIssueLinks({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: issueServiceType,
  });
  // i18n
  const { t } = useTranslation();

  const handleLinkOperations: TLinkOperations = useMemo(
    () => ({
      create: async (data: Partial<TIssueLink>) => {
        try {
          if (!workspaceSlug || !projectId || !issueId) throw new Error("Missing required fields");
          await createLink(data);
          onSuccess?.();
          setToast({
            message: t("links.toasts.created.message"),
            type: TOAST_TYPE.SUCCESS,
            title: t("links.toasts.created.title"),
          });
        } catch (error) {
          setToast({
            message: getLinkErrorMessage(error) ?? t("links.toasts.not_created.message"),
            type: TOAST_TYPE.ERROR,
            title: t("links.toasts.not_created.title"),
          });
          throw error;
        }
      },
      update: async (linkId: string, data: Partial<TIssueLink>) => {
        try {
          if (!workspaceSlug || !projectId || !issueId) throw new Error("Missing required fields");
          await updateLink(linkId, data);
          onSuccess?.();
          setToast({
            message: t("links.toasts.updated.message"),
            type: TOAST_TYPE.SUCCESS,
            title: t("links.toasts.updated.title"),
          });
        } catch (error) {
          setToast({
            message: getLinkErrorMessage(error) ?? t("links.toasts.not_updated.message"),
            type: TOAST_TYPE.ERROR,
            title: t("links.toasts.not_updated.title"),
          });
          throw error;
        }
      },
      remove: async (linkId: string) => {
        try {
          if (!workspaceSlug || !projectId || !issueId) throw new Error("Missing required fields");
          await removeLink(linkId);
          onSuccess?.();
          setToast({
            message: t("links.toasts.removed.message"),
            type: TOAST_TYPE.SUCCESS,
            title: t("links.toasts.removed.title"),
          });
        } catch {
          setToast({
            message: t("links.toasts.not_removed.message"),
            type: TOAST_TYPE.ERROR,
            title: t("links.toasts.not_removed.title"),
          });
        }
      },
    }),
    [workspaceSlug, projectId, issueId, createLink, updateLink, removeLink, t, onSuccess]
  );

  return handleLinkOperations;
};

function getLinkErrorMessage(error: unknown): string | undefined {
  if (!error || typeof error !== "object" || !("data" in error)) return undefined;
  const data = error.data;
  if (data && typeof data === "object" && "error" in data && typeof data.error === "string") return data.error;
  return undefined;
}
