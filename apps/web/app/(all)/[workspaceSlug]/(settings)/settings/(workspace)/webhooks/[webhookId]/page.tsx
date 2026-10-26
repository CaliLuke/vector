/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { EUserPermissions } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWebhook } from "@plane/types";
import { csvDownload } from "@plane/utils/file";
import { LogoSpinner } from "@/components/common/logo-spinner";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { DeleteWebhookModal, WebhookDeleteSection, WebhookForm } from "@/components/web-hooks";
import {
  regenerateWebhookSecretMutationOptions,
  updateWebhookMutationOptions,
  webhookDetailsOptions,
  webhookKeys,
  type IWebhookRecord,
} from "@/queries/webhooks";
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { getCurrentHookAsCSV } from "@/components/web-hooks/utils";
import { WebhooksWorkspaceSettingsHeader } from "../header";

function WebhookDetailsPage(): React.JSX.Element | null {
  const { workspaceSlug, webhookId } = Route.useParams();
  return (
    <WebhookDetailsPageContent
      key={`${workspaceSlug}/${webhookId}`}
      workspaceSlug={workspaceSlug}
      webhookId={webhookId}
    />
  );
}

function WebhookDetailsPageContent({
  workspaceSlug,
  webhookId,
}: {
  workspaceSlug: string;
  webhookId: string;
}): React.JSX.Element | null {
  const [deleteWebhookModal, setDeleteWebhookModal] = useState(false);
  const [revealedSecret, setRevealedSecret] = useState<{ scope: string; value: string } | null>(null);
  const queryClient = useQueryClient();
  const { data: currentWorkspace } = useCurrentWorkspace();
  const {
    can: canWorkspace,
    isPending: permissionPending,
    isError: permissionError,
  } = useWorkspacePermissions(workspaceSlug);
  const isAdmin = canWorkspace([EUserPermissions.ADMIN]);
  const detailsQuery = useQuery(webhookDetailsOptions(workspaceSlug, webhookId, isAdmin));
  const updateWebhook = useMutation(updateWebhookMutationOptions(queryClient, workspaceSlug, webhookId));
  const regenerateSecret = useMutation(regenerateWebhookSecretMutationOptions(queryClient, workspaceSlug, webhookId));
  const secretScope = `${workspaceSlug}/${webhookId}`;
  const secretKey = revealedSecret?.scope === secretScope ? revealedSecret.value : null;
  const regenerateGeneration = useRef(0);
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace.name} - Webhook` : undefined;

  useEffect(() => {
    let generation = ++regenerateGeneration.current;
    const detailKey = webhookKeys.detail(workspaceSlug, webhookId);
    let detailQuery = queryClient.getQueryCache().find({ queryKey: detailKey, exact: true });
    const unsubscribe = queryClient.getQueryCache().subscribe(() => {
      const currentQuery = queryClient.getQueryCache().find({ queryKey: detailKey, exact: true });
      if (currentQuery !== detailQuery) {
        detailQuery = currentQuery;
        generation = ++regenerateGeneration.current;
        setRevealedSecret(null);
      }
    });
    return () => {
      unsubscribe();
      if (regenerateGeneration.current === generation) regenerateGeneration.current += 1;
    };
  }, [queryClient, workspaceSlug, webhookId]);

  const handleUpdateWebhook = async (payload: Partial<IWebhook>) => {
    const generation = regenerateGeneration.current;
    const detailKey = webhookKeys.detail(workspaceSlug, webhookId);
    const detailQuery = queryClient.getQueryCache().find({ queryKey: detailKey, exact: true });
    try {
      const savedWebhook = await updateWebhook.mutateAsync(payload);
      if (
        regenerateGeneration.current !== generation ||
        !detailQuery ||
        queryClient.getQueryCache().find({ queryKey: detailKey, exact: true }) !== detailQuery
      ) {
        return savedWebhook;
      }
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Webhook updated successfully.",
      });
      return savedWebhook;
    } catch (error: unknown) {
      if (
        regenerateGeneration.current !== generation ||
        !detailQuery ||
        queryClient.getQueryCache().find({ queryKey: detailKey, exact: true }) !== detailQuery
      ) {
        throw error;
      }
      const message =
        error && typeof error === "object" && "error" in error && typeof error.error === "string"
          ? error.error
          : "Something went wrong. Please try again.";
      setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message });
      throw error;
    }
  };

  const handleRegenerateSecret = async (): Promise<boolean | "stale"> => {
    const generation = regenerateGeneration.current;
    const submittedScope = secretScope;
    const detailKey = webhookKeys.detail(workspaceSlug, webhookId);
    const detailQuery = queryClient.getQueryCache().find({ queryKey: detailKey, exact: true });
    let downloaded = true;
    await regenerateSecret.mutateAsync({
      exposeSecret: (secret, webhook) => {
        if (
          !secret ||
          regenerateGeneration.current !== generation ||
          submittedScope !== secretScope ||
          !detailQuery ||
          queryClient.getQueryCache().find({ queryKey: detailKey, exact: true }) !== detailQuery
        )
          return;
        setRevealedSecret({ scope: secretScope, value: secret });
        try {
          csvDownload(
            getCurrentHookAsCSV(currentWorkspace ?? null, webhook, secret),
            `webhook-secret-key-${Date.now()}`
          );
        } catch {
          downloaded = false;
        }
      },
    });
    if (
      regenerateGeneration.current !== generation ||
      submittedScope !== secretScope ||
      !detailQuery ||
      queryClient.getQueryCache().find({ queryKey: detailKey, exact: true }) !== detailQuery
    ) {
      return "stale";
    }
    return downloaded;
  };

  if (permissionPending)
    return (
      <div className="grid h-full w-full place-items-center p-4">
        <LogoSpinner />
      </div>
    );

  if (permissionError || !isAdmin)
    return (
      <>
        <PageHead title={pageTitle} />
        <div className="mt-10 flex h-full w-full justify-center p-4">
          <p className="text-13 text-tertiary">You are not authorized to access this page.</p>
        </div>
      </>
    );

  if (detailsQuery.isPending)
    return (
      <div className="grid h-full w-full place-items-center p-4">
        <LogoSpinner />
      </div>
    );

  if (detailsQuery.isError || !detailsQuery.data)
    return (
      <div className="grid h-full w-full place-items-center p-4">
        <div className="space-y-3 text-center" role="alert">
          <p className="text-13 text-danger-primary">Webhook details could not be loaded.</p>
          <button type="button" className="text-13 text-accent-primary" onClick={() => void detailsQuery.refetch()}>
            Retry
          </button>
        </div>
      </div>
    );

  const webhook: IWebhookRecord = detailsQuery.data;
  return (
    <SettingsContentWrapper header={<WebhooksWorkspaceSettingsHeader />}>
      <PageHead title={pageTitle} />
      <DeleteWebhookModal
        key={`${secretScope}/${deleteWebhookModal}`}
        isOpen={deleteWebhookModal}
        workspaceSlug={workspaceSlug}
        webhookId={webhookId}
        onClose={() => setDeleteWebhookModal(false)}
      />
      <div className="w-full space-y-8 overflow-y-auto">
        <div>
          <WebhookForm
            key={secretScope}
            onSubmit={handleUpdateWebhook}
            data={webhook}
            secretKey={secretKey}
            onRegenerate={handleRegenerateSecret}
          />
        </div>
        <WebhookDeleteSection openDeleteModal={() => setDeleteWebhookModal(true)} />
      </div>
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace/webhooks/$webhookId")({
  component: WebhookDetailsPage,
});
