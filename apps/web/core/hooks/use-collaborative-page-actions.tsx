/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect } from "react";
import type { TDocumentEventsServer } from "@plane/editor";
import type { TDocumentEventsClient } from "@plane/editor/lib";
import { DocumentCollaborativeEvents, getServerEventName } from "@plane/editor/lib";
import type { TPage } from "@plane/types";
import { EPageAccess } from "@plane/types";
import { useQueryClient } from "@tanstack/react-query";
import { useOptionalPageRecord } from "@/components/pages/page-record-context";
import { useProjectPageAction } from "@/hooks/use-project-pages";
import { updateProjectPageCache } from "@/queries/pages";

type CollaborativeActionEvent =
  | { type: "sendMessageToServer"; message: TDocumentEventsServer; recursive?: boolean }
  | { type: "receivedMessageFromServer"; message: TDocumentEventsClient };

type Props = {
  page: TPage | undefined;
  workspaceSlug: string;
  projectId?: string;
};

function pageAction(message: TDocumentEventsServer) {
  switch (message) {
    case "lock":
      return "lock";
    case "unlock":
      return "unlock";
    case "archive":
      return "archive";
    case "unarchive":
      return "restore";
    case "make-public":
      return "makePublic";
    case "make-private":
      return "makePrivate";
    default:
      throw new Error("Unsupported page action.");
  }
}

export const useCollaborativePageActions = ({ page, workspaceSlug, projectId: explicitProjectId }: Props) => {
  const pageId = page?.id;
  const projectId = explicitProjectId ?? page?.project_ids?.[0];
  const pageRecord = useOptionalPageRecord();
  const editorRef = pageRecord?.editorRef;
  const client = useQueryClient();
  const actionMutation = useProjectPageAction(workspaceSlug, projectId, pageId);

  const executeCollaborativeAction = useCallback(
    async (event: CollaborativeActionEvent) => {
      const outgoing = event.type === "sendMessageToServer";
      const clientEvent = outgoing ? DocumentCollaborativeEvents[event.message].client : event.message;
      if (!pageId || !projectId) throw new Error("Page action scope is unavailable.");
      if (!outgoing) {
        const changesByEvent: Partial<Record<TDocumentEventsClient, Partial<TPage>>> = {
          locked: { is_locked: true },
          unlocked: { is_locked: false },
          archived: { archived_at: new Date().toISOString() },
          unarchived: { archived_at: null },
          "made-public": { access: EPageAccess.PUBLIC },
          "made-private": { access: EPageAccess.PRIVATE },
        };
        const changes = changesByEvent[clientEvent] ?? {};
        if (pageRecord)
          updateProjectPageCache(client, pageRecord.scope, pageId, (current) => ({ ...current, ...changes }));
        return;
      }
      const serverMessage = getServerEventName(clientEvent);
      await actionMutation.mutateAsync(pageAction(event.message));
      if (serverMessage) editorRef?.emitRealTimeUpdate(serverMessage);
    },
    [actionMutation, client, editorRef, pageId, pageRecord, projectId]
  );

  useEffect(() => {
    const listener = (message: { payload: TDocumentEventsClient }) => {
      void executeCollaborativeAction({ type: "receivedMessageFromServer", message: message.payload }).catch(
        (error) => {
          console.error(error);
        }
      );
    };
    const connection = editorRef?.listenToRealTimeUpdate();
    connection?.on("stateless", listener);
    return () => connection?.off("stateless", listener);
  }, [editorRef, executeCollaborativeAction]);

  return { executeCollaborativeAction };
};
