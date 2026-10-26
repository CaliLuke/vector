/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Link, useRouter } from "@tanstack/react-router";
import { useCallback, useEffect, useMemo } from "react";

// plane types
import { getButtonStyling } from "@plane/propel/button";
import type { TSearchEntityRequestPayload, TWebhookConnectionQueryParams } from "@plane/types";
import { EFileAssetType } from "@plane/types";
// plane ui
// plane utils
import { cn } from "@plane/utils/common";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { PageHead } from "@/components/core/page-title";
import { IssuePeekOverview } from "@/components/issues/peek-overview";
import type { TPageRootConfig, TPageRootHandlers } from "@/components/pages/editor/page-root";
import { PageRoot } from "@/components/pages/editor/page-root";
// hooks
import { useEditorConfig } from "@/hooks/editor";
import { usePagePermissions } from "@/hooks/use-page-permissions";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import { usePageRecord } from "@/components/pages/page-record-context";
import { useCreateProjectPage, useUpdateProjectPageDescription } from "@/hooks/use-project-pages";

// plane web services
import { WorkspaceService } from "@/services/workspace.service";
// services
import { ProjectPageService } from "@/services/page";
const workspaceService = new WorkspaceService();
const projectPageService = new ProjectPageService();

function PageDetailsPage(): React.JSX.Element | null {
  const params = Route.useParams();
  // router
  const router = useRouter();
  const { workspaceSlug, projectId, pageId } = params;
  const { page, pageQuery } = usePageRecord();
  const create = useCreateProjectPage(workspaceSlug, projectId);
  const updatePageDescription = useUpdateProjectPageDescription(workspaceSlug, projectId, pageId);
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  // derived values
  const workspaceId = workspaceDetails?.id ?? "";
  const { canCurrentUserAccessPage } = usePagePermissions(page, workspaceSlug);
  const { id, name } = page ?? {};
  // entity search handler
  const fetchEntityCallback = useCallback(
    async (payload: TSearchEntityRequestPayload) =>
      await workspaceService.searchEntity(workspaceSlug, {
        ...payload,
        project_id: projectId,
      }),
    [projectId, workspaceSlug]
  );
  // editor config
  // fetch page details
  // page root handlers
  const pageRootHandlers: TPageRootHandlers = useMemo(
    () => ({
      create: (payload) => create.mutateAsync(payload),
      fetchDescriptionBinary: async () => {
        if (!id) throw new Error("Page access is unavailable.");
        return await projectPageService.fetchDescriptionBinary(workspaceSlug, projectId, id);
      },
      fetchEntity: fetchEntityCallback,
      getRedirectionLink: (targetPageId) => {
        if (targetPageId) {
          return `/${workspaceSlug}/projects/${projectId}/pages/${targetPageId}`;
        } else {
          return `/${workspaceSlug}/projects/${projectId}/pages`;
        }
      },
      updateDescription: (document) => updatePageDescription.mutateAsync(document),
    }),
    [create, fetchEntityCallback, id, updatePageDescription, workspaceSlug, projectId]
  );
  const fileHandler = useEditorConfig({
    projectId,
    fileAsset: { entity_identifier: id ?? "", entity_type: EFileAssetType.PAGE_DESCRIPTION },
    workspaceId,
    workspaceSlug,
  });
  const pageRootConfig: TPageRootConfig = { fileHandler };

  const webhookConnectionParams: TWebhookConnectionQueryParams = useMemo(
    () => ({
      documentType: "project_page",
      projectId,
      workspaceSlug,
    }),
    [projectId, workspaceSlug]
  );

  useEffect(() => {
    if (page?.deleted_at) {
      void router.navigate({ href: pageRootHandlers.getRedirectionLink() });
    }
  }, [page?.deleted_at, router, pageRootHandlers]);

  if (pageQuery.isError || pageQuery.access.isError)
    return (
      <div className="flex h-full w-full flex-col items-center justify-center">
        <h3 className="text-center text-16 font-semibold">Page could not be loaded</h3>
        <p className="mt-3 text-center text-13 text-secondary">Try again when your connection is available.</p>
        <button
          type="button"
          className={cn(getButtonStyling("secondary", "base"), "mt-5")}
          onClick={() => void pageQuery.refetch()}
        >
          Try again
        </button>
      </div>
    );

  if (pageQuery.access.isDenied || (page && !canCurrentUserAccessPage))
    return (
      <div className="flex h-full w-full flex-col items-center justify-center">
        <h3 className="text-center text-16 font-semibold">Page not found</h3>
        <p className="mt-3 text-center text-13 text-secondary">
          The page you are trying to access doesn{"'"}t exist or you don{"'"}t have permission to view it.
        </p>
        <Link
          to="/$workspaceSlug/projects/$projectId/pages/"
          params={{ workspaceSlug: workspaceSlug, projectId: projectId }}
          className={cn(getButtonStyling("secondary", "base"), "mt-5")}
        >
          View other Pages
        </Link>
      </div>
    );

  if (!page)
    return (
      <div className="grid size-full place-items-center">
        <LogoSpinner />
      </div>
    );

  return (
    <>
      <PageHead title={name} />
      <div className="flex h-full flex-col justify-between">
        <div className="relative flex h-full w-full flex-shrink-0 flex-col overflow-hidden">
          <PageRoot
            config={pageRootConfig}
            handlers={pageRootHandlers}
            page={page}
            webhookConnectionParams={webhookConnectionParams}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
          />
          <IssuePeekOverview />
        </div>
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/pages/$pageId/")({
  component: PageDetailsPage,
});
