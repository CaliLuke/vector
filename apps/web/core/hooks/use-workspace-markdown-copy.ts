/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import type { EditorRefApi } from "@plane/editor";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";

type MarkdownCopyEditor = Pick<EditorRefApi, "copyMarkdownToClipboard">;

export function useWorkspaceMarkdownCopy(workspaceSlug: string) {
  const { query, isLoading, isError, retry } = useWorkspaceMemberRoster(workspaceSlug);
  const isReady = query.isSuccess && !isLoading && !isError;
  const state = isReady ? "ready" : isError ? "error" : "loading";
  const handleAction = useCallback(
    (getEditor: () => MarkdownCopyEditor | null, onCopied: () => void) => {
      if (state === "error") {
        void retry();
        return;
      }
      if (state !== "ready") return;
      const editor = getEditor();
      if (!editor) return;
      try {
        editor.copyMarkdownToClipboard();
      } catch {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Copy failed",
          message: "The markdown could not be copied. Please try again.",
        });
        return;
      }
      onCopied();
    },
    [retry, state]
  );

  return {
    state,
    actionLabel:
      state === "ready"
        ? "Copy markdown"
        : state === "error"
          ? "Retry loading workspace members"
          : "Loading workspace members…",
    actionDisabled: state === "loading",
    handleAction,
    statusRole: state === "error" ? ("alert" as const) : ("status" as const),
    statusMessage:
      state === "error"
        ? "Workspace members could not be loaded. Retry before copying markdown."
        : state === "loading" && isLoading
          ? "Loading workspace members before copying markdown…"
          : undefined,
  };
}
