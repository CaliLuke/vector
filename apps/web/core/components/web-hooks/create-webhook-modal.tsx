/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { useParams } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspace } from "@plane/types";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { csvDownload } from "@plane/utils/file";
import useKeypress from "@/hooks/use-keypress";
import { createWebhookMutationOptions, webhookKeys, type IWebhookRecord } from "@/queries/webhooks";
import { WebhookForm } from "./form";
import { GeneratedHookDetails } from "./generated-hook-details";
import { getCurrentHookAsCSV } from "./utils";

type Props = {
  currentWorkspace: IWorkspace | null;
  isOpen: boolean;
  onClose: () => void;
};

export function CreateWebhookModal({ isOpen, onClose, currentWorkspace }: Props) {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const createWebhook = useMutation(createWebhookMutationOptions(queryClient, workspaceSlug));
  const [generatedWebhookState, setGeneratedWebhook] = useState<{ scope: string; webhook: IWebhookRecord } | null>(
    null
  );
  const [secretState, setSecretKey] = useState<{ scope: string; value: string } | null>(null);
  const scope = workspaceSlug;
  const generatedWebhook = generatedWebhookState?.scope === scope ? generatedWebhookState.webhook : null;
  const secretKey = secretState?.scope === scope ? secretState.value : null;
  const flowVersion = useRef(0);

  useEffect(() => {
    const listKey = webhookKeys.list(scope);
    let listQuery = queryClient.getQueryCache().find({ queryKey: listKey, exact: true });

    const unsubscribe = queryClient.getQueryCache().subscribe(() => {
      const currentQuery = queryClient.getQueryCache().find({ queryKey: listKey, exact: true });
      if (currentQuery === listQuery) return;
      listQuery = currentQuery;
      flowVersion.current += 1;
      setSecretKey(null);
      setGeneratedWebhook(null);
    });
    return () => {
      unsubscribe();
      flowVersion.current += 1;
    };
  }, [queryClient, scope]);

  const handleCreateWebhook = async (payload: Partial<IWebhookRecord>) => {
    const submittedScope = scope;
    const listKey = webhookKeys.list(workspaceSlug);
    const listQuery = queryClient.getQueryCache().find({ queryKey: listKey, exact: true });
    const flowAtSubmit = flowVersion.current;
    try {
      const webhook = await createWebhook.mutateAsync({
        data: payload,
        exposeSecret: (secret, createdWebhook) => {
          if (
            flowVersion.current !== flowAtSubmit ||
            !listQuery ||
            queryClient.getQueryCache().find({ queryKey: listKey, exact: true }) !== listQuery
          )
            return;

          setSecretKey(secret ? { scope: submittedScope, value: secret } : null);
          if (!secret) return;
          try {
            csvDownload(
              getCurrentHookAsCSV(currentWorkspace, createdWebhook, secret),
              `webhook-secret-key-${Date.now()}`
            );
          } catch {
            setToast({
              type: TOAST_TYPE.ERROR,
              title: "Webhook created",
              message: "The CSV download failed. Copy the secret from this dialog.",
            });
          }
        },
      });
      if (
        flowVersion.current !== flowAtSubmit ||
        !listQuery ||
        queryClient.getQueryCache().find({ queryKey: listKey, exact: true }) !== listQuery
      )
        return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("workspace_settings.settings.webhooks.toasts.created.title"),
        message: t("workspace_settings.settings.webhooks.toasts.created.message"),
      });
      setGeneratedWebhook({ scope: submittedScope, webhook });
    } catch (error: unknown) {
      if (flowVersion.current === flowAtSubmit) {
        const responseError = error as { error?: unknown };
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("workspace_settings.settings.webhooks.toasts.not_created.title"),
          message:
            (typeof responseError?.error === "string" && responseError.error) ||
            t("workspace_settings.settings.webhooks.toasts.not_created.message"),
        });
      }
      throw error;
    }
  };

  const handleClose = () => {
    flowVersion.current += 1;
    setSecretKey(null);
    setGeneratedWebhook(null);
    onClose();
  };

  useKeypress("Escape", () => {
    if (isOpen && !generatedWebhook && !createWebhook.isPending) handleClose();
  });

  return (
    <ModalCore isOpen={isOpen} position={EModalPosition.TOP} width={EModalWidth.XXL} className="p-4 pb-0">
      {!generatedWebhook ? (
        <WebhookForm onSubmit={handleCreateWebhook} handleClose={handleClose} />
      ) : (
        <GeneratedHookDetails webhookDetails={generatedWebhook} secretKey={secretKey} handleClose={handleClose} />
      )}
    </ModalCore>
  );
}
