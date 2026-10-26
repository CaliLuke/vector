/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { setPromiseToast, TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// hooks
import { useIssueAttachments } from "@/hooks/use-issue-resources";
import type { TAttachmentUploadStatus } from "@/hooks/use-issue-resources";

export type TAttachmentOperations = {
  create: (file: File) => Promise<void>;
  remove: (attachmentId: string) => Promise<void>;
};

export type TAttachmentSnapshot = {
  uploadStatus: readonly TAttachmentUploadStatus[];
  attachments: readonly import("@plane/types").TIssueAttachment[] | undefined;
  isError: boolean;
  isDenied: boolean;
  retry: () => Promise<unknown>;
};

export type TAttachmentHelpers = {
  uploadScopeKey: string;
  operations: TAttachmentOperations;
  snapshot: TAttachmentSnapshot;
};

export const useAttachmentOperations = (
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  issueServiceType: TIssueServiceType = EIssueServiceType.ISSUES
): TAttachmentHelpers => {
  const resources = useIssueAttachments({ workspaceSlug, projectId, issueId, serviceType: issueServiceType });
  const { create, remove, uploadStatus, attachments, query, isError, retry } = resources;

  const attachmentOperations: TAttachmentOperations = useMemo(
    () => ({
      create: async (file) => {
        if (!workspaceSlug || !projectId || !issueId) throw new Error("Missing required fields");
        const attachmentUploadPromise = create(file);
        setPromiseToast(attachmentUploadPromise, {
          loading: "Uploading attachment...",
          success: {
            title: "Attachment uploaded",
            message: () => "The attachment has been successfully uploaded",
          },
          error: {
            title: "Attachment not uploaded",
            message: () => "The attachment could not be uploaded",
          },
        });

        await attachmentUploadPromise;
      },
      remove: async (attachmentId) => {
        try {
          if (!workspaceSlug || !projectId || !issueId) throw new Error("Missing required fields");
          await remove(attachmentId);
          setToast({
            message: "The attachment has been successfully removed",
            type: TOAST_TYPE.SUCCESS,
            title: "Attachment removed",
          });
        } catch (_error) {
          setToast({
            message: "The Attachment could not be removed",
            type: TOAST_TYPE.ERROR,
            title: "Attachment not removed",
          });
          throw _error;
        }
      },
    }),
    [create, issueId, projectId, remove, workspaceSlug]
  );

  return {
    uploadScopeKey: resources.uploadScopeKey,
    operations: attachmentOperations,
    snapshot: {
      uploadStatus,
      attachments,
      isError: query.isError || isError,
      isDenied: resources.isDenied,
      retry: () => (isError ? retry() : query.refetch()),
    },
  };
};
