/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useLayoutEffect, useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { EditorRefApi, CollaborationState } from "@plane/editor";
// plane editor
import { convertBinaryDataToBase64String, getBinaryDataFromDocumentEditorHTMLString } from "@plane/editor";
// plane types
import type { EUserWorkspaceRoles, IUserProjectsRole, TDocumentPayload, TPage } from "@plane/types";
// hooks
import useAutoSave from "@/hooks/use-auto-save";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { captureIdentityScope, isIdentityScopeVerified } from "@/queries/identity";
import { getPagePermissions } from "@/helpers/page-permissions";

type TFallbackScope = {
  currentUserId: string | undefined;
  workspaceSlug: string;
  roles: IUserProjectsRole | undefined;
  workspaceRole: EUserWorkspaceRoles | null | undefined;
  page: TPage;
  editorRef: React.RefObject<EditorRefApi | null>;
};

function isFallbackScopeCurrent(args: {
  client: QueryClient;
  identityScope: NonNullable<ReturnType<typeof captureIdentityScope>>;
  initiatingScope: TFallbackScope;
  latestScope: TFallbackScope | undefined;
  page: TPage;
  pageId: string | undefined;
  editorRef: React.RefObject<EditorRefApi | null>;
  editor: EditorRefApi;
}) {
  const { client, identityScope, initiatingScope, latestScope, page, pageId, editorRef, editor } = args;
  return Boolean(
    isIdentityScopeVerified(client, identityScope) &&
    latestScope &&
    latestScope.currentUserId === initiatingScope.currentUserId &&
    latestScope.workspaceSlug === initiatingScope.workspaceSlug &&
    latestScope.page === page &&
    latestScope.editorRef === editorRef &&
    page.id === pageId &&
    editorRef.current === editor
  );
}

function canPersistFallbackDescription(page: TPage, scope: TFallbackScope) {
  return Boolean(
    scope.roles &&
    scope.workspaceRole !== null &&
    scope.workspaceRole !== undefined &&
    getPagePermissions(page, scope.currentUserId, scope.roles, scope.workspaceRole).isContentEditable
  );
}

function getFallbackDescription(encoded: ArrayBuffer, page: TPage) {
  return encoded.byteLength > 0
    ? new Uint8Array(encoded)
    : getBinaryDataFromDocumentEditorHTMLString(page.description_html ?? "<p></p>", page.name);
}

type TArgs = {
  editorRef: React.RefObject<EditorRefApi | null>;
  fetchPageDescription: () => Promise<ArrayBuffer>;
  collaborationState: CollaborationState | null;
  updatePageDescription: (data: TDocumentPayload) => Promise<void>;
  page: TPage;
  workspaceSlug: string;
};

export const usePageFallback = (args: TArgs) => {
  const { editorRef, fetchPageDescription, collaborationState, updatePageDescription, page, workspaceSlug } = args;
  const currentUserId = useCurrentUserId();
  const queryClient = useQueryClient();
  const { roles, workspaceRole } = useWorkspaceProjectRoles(workspaceSlug);
  const latestScopeRef = useRef<TFallbackScope | undefined>(undefined);
  useLayoutEffect(() => {
    latestScopeRef.current = { currentUserId, workspaceSlug, roles, workspaceRole, page, editorRef };
    return () => {
      latestScopeRef.current = undefined;
    };
  }, [currentUserId, workspaceSlug, roles, workspaceRole, page, editorRef]);
  const hasShownFallbackToast = useRef(false);
  const isFallbackUpdateInFlight = useRef(false);

  // Derive connection failure from collaboration state
  const hasConnectionFailed = collaborationState?.stage.kind === "disconnected";

  const { mutateAsync: recoverDescription, isPending: isFetchingFallbackBinary } = useMutation({
    networkMode: "always",
    retry: false,
    mutationFn: async ({
      editor,
      initiatingScope,
      identityScope,
      fetchDescription,
      persistDescription,
    }: {
      fetchDescription: TArgs["fetchPageDescription"];
      persistDescription: TArgs["updatePageDescription"];
      editor: EditorRefApi;
      initiatingScope: TFallbackScope;
      identityScope: NonNullable<ReturnType<typeof captureIdentityScope>>;
    }) => {
      const { page: initiatingPage, editorRef: initiatingEditorRef } = initiatingScope;
      const initiatingPageId = initiatingPage.id;
      const isCurrentScope = (latestScope: TFallbackScope | undefined) =>
        isFallbackScopeCurrent({
          client: queryClient,
          identityScope,
          initiatingScope,
          latestScope,
          page: initiatingPage,
          pageId: initiatingPageId,
          editorRef: initiatingEditorRef,
          editor,
        });

      // Show toast notification when fallback mechanism kicks in (only once)
      if (!hasShownFallbackToast.current) {
        console.warn("Websocket Connection lost, your changes are being saved using backup mechanism.");
        hasShownFallbackToast.current = true;
      }

      try {
        const latestEncodedDescription = await fetchDescription();
        const latestScope = latestScopeRef.current;
        if (!isCurrentScope(latestScope)) return;
        editor.setProviderDocument(getFallbackDescription(latestEncodedDescription, initiatingPage));
        const { binary, html, json } = editor.getDocument();
        // Hydrate read-only pages from the fallback document, but don't persist it.
        // Read the live permission after the fetch so a lock/archive transition
        // while the request is in flight cannot trigger a write.
        if (!binary || !json || !latestScope || !canPersistFallbackDescription(initiatingPage, latestScope)) return;
        const encodedBinary = convertBinaryDataToBase64String(binary);

        await persistDescription({
          description_binary: encodedBinary,
          description_html: html,
          description_json: json,
        });
      } catch (error: unknown) {
        console.error(error);
      } finally {
        isFallbackUpdateInFlight.current = false;
      }
    },
  });

  const handleUpdateDescription = useCallback(async () => {
    if (!hasConnectionFailed) return;
    const editor = editorRef.current;
    if (!editor || isFallbackUpdateInFlight.current) return;
    const initiatingScope = latestScopeRef.current;
    if (!initiatingScope || initiatingScope.page !== page) return;
    const identityScope = captureIdentityScope(queryClient, initiatingScope.currentUserId);
    if (!identityScope || !isIdentityScopeVerified(queryClient, identityScope)) return;
    isFallbackUpdateInFlight.current = true;
    await recoverDescription({
      editor,
      initiatingScope,
      identityScope,
      fetchDescription: fetchPageDescription,
      persistDescription: updatePageDescription,
    });
  }, [
    editorRef,
    hasConnectionFailed,
    queryClient,
    recoverDescription,
    fetchPageDescription,
    updatePageDescription,
    page,
  ]);

  useEffect(() => {
    if (hasConnectionFailed) {
      void handleUpdateDescription();
    } else {
      // Reset toast flag when connection is restored
      hasShownFallbackToast.current = false;
    }
  }, [handleUpdateDescription, hasConnectionFailed]);

  useAutoSave(() => {
    void handleUpdateDescription();
  });

  return { isFetchingFallbackBinary };
};
