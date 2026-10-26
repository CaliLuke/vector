/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useRef, useState } from "react";
// plane imports
import type { CollaborationState, EditorRefApi } from "@plane/editor";
import type { TDocumentPayload, TPage, TWebhookConnectionQueryParams } from "@plane/types";
// hooks
import { usePageFallback } from "@/hooks/use-page-fallback";
import { usePagePermissions } from "@/hooks/use-page-permissions";
import type { PageUpdateHandler, TCustomEventHandlers } from "@/hooks/use-realtime-page-events";
import { usePagesPaneExtensions, useExtendedEditorProps } from "@/hooks/pages";
import { usePageRecord } from "@/components/pages/page-record-context";
// store
// local imports
import { PageNavigationPaneRoot } from "../navigation-pane";
import { PageVersionsOverlay } from "../version";
import { PagesVersionEditor } from "../version/editor";
import { ContentLimitBanner } from "./content-limit-banner";
import { PageEditorBody } from "./editor-body";
import type { TEditorBodyConfig, TEditorBodyHandlers } from "./editor-body";
import { PageEditorToolbarRoot } from "./toolbar";

export type TPageRootHandlers = {
  create: (payload: Partial<TPage>) => Promise<Partial<TPage> | undefined>;
  fetchDescriptionBinary: () => Promise<ArrayBuffer>;
  updateDescription: (document: TDocumentPayload) => Promise<void>;
} & TEditorBodyHandlers;

export type TPageRootConfig = TEditorBodyConfig;

type TPageRootProps = {
  config: TPageRootConfig;
  handlers: TPageRootHandlers;
  page: TPage;
  webhookConnectionParams: TWebhookConnectionQueryParams;
  projectId?: string;
  workspaceSlug: string;
  customRealtimeEventHandlers?: TCustomEventHandlers;
};

export function PageRoot(props: TPageRootProps) {
  const { config, handlers, page, projectId, webhookConnectionParams, workspaceSlug, customRealtimeEventHandlers } =
    props;
  // states
  const [collaborationState, setCollaborationState] = useState<CollaborationState>({
    stage: { kind: "connecting" },
    isServerSynced: false,
    isServerDisconnected: false,
  });
  const [showContentTooLargeBanner, setShowContentTooLargeBanner] = useState(false);
  const { setEditorRef } = usePageRecord();
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const { isContentEditable } = usePagePermissions(page, workspaceSlug);
  // derived values
  // page fallback
  const { isFetchingFallbackBinary } = usePageFallback({
    editorRef,
    fetchPageDescription: handlers.fetchDescriptionBinary,
    page,
    workspaceSlug,
    collaborationState,
    updatePageDescription: handlers.updateDescription,
  });

  const handleEditorReady = useCallback(
    (ready: boolean) => setEditorRef(ready ? editorRef.current : null),
    [setEditorRef]
  );

  // Get extensions and navigation logic from hook
  const {
    editorExtensionHandlers,
    navigationPaneExtensions,
    handleOpenNavigationPane,
    handleCloseNavigationPane,
    isNavigationPaneOpen,
  } = usePagesPaneExtensions({
    page,
    editorRef,
  });

  // Type-safe error handler for content too large errors
  const errorHandler: PageUpdateHandler<"error"> = (params) => {
    const { data } = params;

    // Check if it's content too large error
    if (data.error_code === "content_too_large") {
      setShowContentTooLargeBanner(true);
    }

    // Call original error handler if exists
    customRealtimeEventHandlers?.error?.(params);
  };

  const mergedCustomEventHandlers: TCustomEventHandlers = {
    ...customRealtimeEventHandlers,
    error: errorHandler,
  };

  // Get extended editor extensions configuration
  const extendedEditorProps = useExtendedEditorProps({
    workspaceSlug,
    page,
    fetchEntity: handlers.fetchEntity,
    getRedirectionLink: handlers.getRedirectionLink,
    extensionHandlers: editorExtensionHandlers,
    projectId,
  });

  const handleRestoreVersion = useCallback(
    async (descriptionHTML: string) => {
      editorRef.current?.clearEditor();
      editorRef.current?.setEditorValue(descriptionHTML);
    },
    [editorRef]
  );

  // reset editor ref on unmount
  useEffect(
    () => () => {
      setEditorRef(null);
    },
    [setEditorRef]
  );

  return (
    <div className="relative flex size-full overflow-hidden transition-all duration-300 ease-in-out">
      <div className="flex size-full flex-col overflow-hidden">
        <PageVersionsOverlay
          editorComponent={PagesVersionEditor}

          handleRestore={handleRestoreVersion}
          pageId={page.id ?? ""}
          restoreEnabled={isContentEditable}
        />
        <PageEditorToolbarRoot
          handleOpenNavigationPane={handleOpenNavigationPane}
          isNavigationPaneOpen={isNavigationPaneOpen}
          page={page}
          workspaceSlug={workspaceSlug}
        />
        {showContentTooLargeBanner && <ContentLimitBanner className="px-page-x" />}
        <PageEditorBody
          config={config}
          customRealtimeEventHandlers={mergedCustomEventHandlers}
          editorForwardRef={editorRef}
          handleEditorReady={handleEditorReady}
          handleOpenNavigationPane={handleOpenNavigationPane}
          handlers={handlers}
          isNavigationPaneOpen={isNavigationPaneOpen}
          page={page}
          projectId={projectId}
          webhookConnectionParams={webhookConnectionParams}
          workspaceSlug={workspaceSlug}
          extendedEditorProps={extendedEditorProps}
          isFetchingFallbackBinary={isFetchingFallbackBinary}
          onCollaborationStateChange={setCollaborationState}
        />
      </div>
      <PageNavigationPaneRoot
        handleClose={handleCloseNavigationPane}
        isNavigationPaneOpen={isNavigationPaneOpen}
        page={page}

        extensions={navigationPaneExtensions}
      />
    </div>
  );
}
