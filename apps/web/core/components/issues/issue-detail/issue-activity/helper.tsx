/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { discussionOwnerKey } from "@/queries/issue-discussion";
import type { CommentActions } from "@/components/comments/types";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { copyUrlToClipboard } from "@plane/utils/string";
import { generateWorkItemLink } from "@plane/utils/work-item";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import type { useIssueDiscussion } from "@/hooks/use-issue-discussion";

type Discussion = ReturnType<typeof useIssueDiscussion>;
export function useWorkItemCommentOperations(
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  discussion: Discussion
): CommentActions {
  const { t } = useTranslation();
  const detail = useWorkItemDetail(workspaceSlug, projectId, issueId, discussion.scope.serviceType);
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const current = useCommittedScope(discussionOwnerKey(discussion.scope));
  const notify = async <T extends { captured: { isCurrent: () => boolean } }>(
    write: () => Promise<T>,
    success: string,
    failure: string
  ) => {
    try {
      const result = await write();
      if (current() && result.captured.isCurrent())
        setToast({ title: t("common.success"), type: TOAST_TYPE.SUCCESS, message: success });
      return result;
    } catch (error) {
      if (current() && discussion.isCurrent())
        setToast({ title: t("common.error.label"), type: TOAST_TYPE.ERROR, message: failure });
      throw error;
    }
  };
  const react = async (commentId: string, reaction: string, kind: "toggle") => {
    await notify(
      () => discussion.reactions.mutateAsync({ commentId, reaction, kind }),
      "Reaction saved successfully",
      "Could not save reaction"
    );
  };
  return {
    isCurrent: () => current() && discussion.isCurrent(),
    isVerified: () => current() && discussion.isVerified(),
    copyCommentLink: (id) => {
      const item = detail.issue;
      if (!item) return;
      const link = generateWorkItemLink({
        workspaceSlug,
        projectId: item.project_id,
        issueId,
        projectIdentifier: projectRows?.find((project) => project.id === projectId)?.identifier,
        sequenceId: item.sequence_id,
      });
      void copyUrlToClipboard(`${link}#comment-${id}`)
        .then(() => {
          if (current() && discussion.isCurrent())
            setToast({
              title: t("common.success"),
              type: TOAST_TYPE.SUCCESS,
              message: t("issue.comments.copy_link.success"),
            });
          return undefined;
        })
        .catch(() => {
          if (current() && discussion.isCurrent())
            setToast({
              title: t("common.error.label"),
              type: TOAST_TYPE.ERROR,
              message: t("issue.comments.copy_link.error"),
            });
          return undefined;
        });
    },
    createComment: async (data) =>
      (
        await notify(
          () => discussion.commentMutation.mutateAsync({ kind: "create", data }),
          t("issue.comments.create.success"),
          t("issue.comments.create.error")
        )
      ).comment,
    updateComment: async (commentId, data) => {
      await notify(
        () => discussion.commentMutation.mutateAsync({ kind: "update", commentId, data }),
        t("issue.comments.update.success"),
        t("issue.comments.update.error")
      );
    },
    removeComment: async (commentId) => {
      await notify(
        () => discussion.commentMutation.mutateAsync({ kind: "delete", commentId }),
        t("issue.comments.remove.success"),
        t("issue.comments.remove.error")
      );
    },
    react: (id, reaction) => react(id, reaction, "toggle"),
  };
}
