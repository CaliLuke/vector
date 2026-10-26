/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { TFileEntityInfo } from "@plane/types";
import { FileService } from "@/services/file.service";

const fileService = new FileService();

type AssetScope = {
  workspaceSlug: string;
  projectId?: string;
  data: TFileEntityInfo;
};
type Upload = AssetScope & { blockId: string; file: File };
type Duplicate = AssetScope & { assetId: string };

export function useEditorUploads(scopeKey: string) {
  const requests = useRef(new Map<string | symbol, { controller: AbortController; scopeKey: string }>());
  const [uploadState, setUploadState] = useState<{ scopeKey: string; progress: Record<string, number> }>({
    scopeKey,
    progress: {},
  });
  const progress = uploadState.scopeKey === scopeKey ? uploadState.progress : {};
  const setProgress = useCallback(
    (update: (current: Record<string, number>) => Record<string, number>) => {
      setUploadState((current) => ({
        scopeKey,
        progress: update(current.scopeKey === scopeKey ? current.progress : {}),
      }));
    },
    [scopeKey]
  );
  useEffect(() => {
    const active = requests.current;
    return () => {
      active.forEach((request, key) => {
        if (request.scopeKey === scopeKey) {
          request.controller.abort();
          active.delete(key);
        }
      });
    };
  }, [scopeKey]);

  const { mutateAsync: upload } = useMutation({
    retry: false,
    mutationFn: async ({
      blockId,
      file,
      data,
      projectId,
      workspaceSlug,
      controller,
    }: Upload & { controller: AbortController }) => {
      const { signal } = controller;
      signal.throwIfAborted();
      const onProgress = (event: { progress?: number }) => {
        if (signal.aborted || requests.current.get(blockId)?.controller !== controller) return;
        setProgress((current) => ({ ...current, [blockId]: Math.round((event.progress ?? 0) * 100) }));
      };
      const result = projectId
        ? await fileService.uploadProjectAsset(workspaceSlug, projectId, data, file, onProgress, signal)
        : await fileService.uploadWorkspaceAsset(workspaceSlug, data, file, onProgress, signal);
      signal.throwIfAborted();
      return result.asset_id;
    },
  });
  const { mutateAsync: duplicate } = useMutation({
    retry: false,
    mutationFn: async ({
      assetId,
      data,
      projectId,
      workspaceSlug,
      controller,
    }: Duplicate & { controller: AbortController }) => {
      controller.signal.throwIfAborted();
      const result = await fileService.duplicateAsset(
        workspaceSlug,
        assetId,
        {
          entity_id: data.entity_identifier || undefined,
          entity_type: data.entity_type,
          project_id: projectId,
        },
        controller.signal
      );
      controller.signal.throwIfAborted();
      return result.asset_id;
    },
  });

  const uploadAsset = async (args: Upload) => {
    const { blockId } = args;
    requests.current.get(blockId)?.controller.abort();
    const controller = new AbortController();
    requests.current.set(blockId, { controller, scopeKey });
    setProgress((current) => ({ ...current, [blockId]: 0 }));
    try {
      return await upload({ ...args, controller });
    } finally {
      if (requests.current.get(blockId)?.controller === controller) {
        requests.current.delete(blockId);
        setProgress((current) => {
          const next = { ...current };
          delete next[blockId];
          return next;
        });
      }
    }
  };

  const duplicateAsset = async (args: Duplicate) => {
    const key = Symbol();
    const controller = new AbortController();
    requests.current.set(key, { controller, scopeKey });
    try {
      return await duplicate({ ...args, controller });
    } finally {
      requests.current.delete(key);
    }
  };

  const cancelUploads = useCallback(() => {
    requests.current.forEach(({ controller }) => controller.abort());
    requests.current.clear();
    setProgress(() => ({}));
  }, [setProgress]);

  return { progress, uploadAsset, duplicateAsset, cancelUploads };
}
