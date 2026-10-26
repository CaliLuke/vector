/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { EUserPermissions } from "@plane/types";
import type { TIssueComment, TIssueCommentReaction, TIssueReaction } from "@plane/types";
import { IssueCommentService } from "@/services/issue/issue_comment.service";
import { IssueReactionService } from "@/services/issue/issue_reaction.service";
import { captureDiscussionWrite, discussionKey, refreshIssueActivity } from "./issue-discussion";
import type { DiscussionScope, DiscussionCollection, IssueActivityRefreshOwner } from "./issue-discussion";

type CommentWrite =
  | { kind: "create"; data: Partial<TIssueComment> }
  | { kind: "update"; commentId: string; data: Partial<TIssueComment> }
  | { kind: "delete"; commentId: string };
type ReactionChange = { kind: "add" | "remove"; reaction: string; commentId?: string };
type ReactionWrite = Omit<ReactionChange, "kind"> & { kind: "add" | "remove" | "toggle" };
type SavedReaction =
  | { kind: "comment"; commentId: string; saved: TIssueCommentReaction | undefined }
  | { kind: "issue"; saved: TIssueReaction | undefined };
function requireComment(client: QueryClient, scope: DiscussionScope, id: string) {
  const comment = client
    .getQueryData<TIssueComment[]>(discussionKey(scope, "comments"))
    ?.find((item) => item.id === id);
  if (!comment || comment.issue !== scope.issueId || comment.project !== scope.projectId)
    throw new Error("Reload this comment before changing it.");
  return comment;
}
function mutationScope(client: QueryClient, scope: DiscussionScope, collection: DiscussionCollection) {
  const captured = captureDiscussionWrite(client, scope, collection);
  const { isVerified, isCurrent } = captured;
  const key = discussionKey(scope, collection);
  const check = () => {
    if (!isVerified()) throw new Error("Work item discussion access changed.");
  };
  const reconcile = async (apply: () => void) => {
    if (!isCurrent()) return;
    await client.cancelQueries({ queryKey: key, exact: true });
    if (isCurrent()) {
      apply();
      reconcileActivity(client, scope, { accountId: scope.accountId, generation: scope.generation, isCurrent });
    }
  };
  return { captured: { isVerified, isCurrent }, check, key, scopeId: JSON.stringify(key.slice(0, 12)), reconcile };
}
function reconcileActivity(client: QueryClient, scope: DiscussionScope, owner: IssueActivityRefreshOwner) {
  void refreshIssueActivity(client, scope, owner).catch((error) =>
    console.error("Could not refresh work item activity", error)
  );
}
async function sendComment(scope: DiscussionScope, input: CommentWrite) {
  const service = new IssueCommentService(scope.serviceType);
  if (input.kind === "create")
    return service.createIssueComment(scope.workspaceSlug, scope.projectId, scope.issueId, input.data);
  if (input.kind === "update")
    return service.patchIssueComment(scope.workspaceSlug, scope.projectId, scope.issueId, input.commentId, input.data);
  await service.deleteIssueComment(scope.workspaceSlug, scope.projectId, scope.issueId, input.commentId);
  return undefined;
}
function applyComment(
  client: QueryClient,
  scope: DiscussionScope,
  input: CommentWrite,
  comment: TIssueComment | undefined
) {
  const key = discussionKey(scope, "comments");
  if (input.kind === "delete") {
    client.setQueryData<TIssueComment[]>(key, (rows) => rows?.filter((row) => row.id !== input.commentId));
    return;
  }
  if (
    !comment?.id ||
    comment.project !== scope.projectId ||
    comment.issue !== scope.issueId ||
    (input.kind === "update" && comment.id !== input.commentId)
  )
    throw new Error("The saved comment response did not match this work item.");
  client.setQueryData<TIssueComment[]>(key, (rows = []) => [...rows.filter((row) => row.id !== comment.id), comment]);
}
export function commentMutationOptions(client: QueryClient, scope: DiscussionScope) {
  const { captured, check, key, scopeId, reconcile } = mutationScope(client, scope, "comments");
  return mutationOptions({
    mutationKey: [...key, "write"],
    scope: { id: scopeId },
    retry: false,
    mutationFn: async (input: CommentWrite) => {
      check();
      if (input.kind !== "create") {
        const comment = requireComment(client, scope, input.commentId);
        if (
          comment.created_by !== scope.accountId &&
          comment.actor !== scope.accountId &&
          scope.projectRole !== EUserPermissions.ADMIN
        )
          throw new Error("You cannot change this comment.");
      }
      await client.cancelQueries({ queryKey: key, exact: true });
      check();
      const comment = await sendComment(scope, input);
      // A received durable result stays successful when its UI/query scope has retired.
      await reconcile(() => applyComment(client, scope, input, comment));
      return { comment, captured };
    },
  });
}
function resolveReaction(client: QueryClient, scope: DiscussionScope, input: ReactionWrite): ReactionChange {
  if (input.kind !== "toggle") return { ...input, kind: input.kind };
  const rows = input.commentId
    ? requireComment(client, scope, input.commentId).comment_reactions
    : (client.getQueryData<TIssueReaction[]>(discussionKey(scope, "reactions")) ?? []);
  return {
    ...input,
    kind: rows.some((row) => row.actor === scope.accountId && row.reaction === input.reaction) ? "remove" : "add",
  };
}
async function sendReaction(scope: DiscussionScope, input: ReactionChange): Promise<SavedReaction> {
  const service = new IssueReactionService(scope.serviceType);
  if (input.commentId) {
    const saved =
      input.kind === "add"
        ? await service.createIssueCommentReaction(scope.workspaceSlug, scope.projectId, input.commentId, {
            reaction: input.reaction,
          })
        : await service.deleteIssueCommentReaction(
            scope.workspaceSlug,
            scope.projectId,
            input.commentId,
            input.reaction
          );
    return { kind: "comment", commentId: input.commentId, saved: saved ?? undefined };
  }
  const saved =
    input.kind === "add"
      ? await service.createIssueReaction(scope.workspaceSlug, scope.projectId, scope.issueId, {
          reaction: input.reaction,
        })
      : await service.deleteIssueReaction(scope.workspaceSlug, scope.projectId, scope.issueId, input.reaction);
  return { kind: "issue", saved: saved ?? undefined };
}
function validateReaction(scope: DiscussionScope, input: ReactionChange, receipt: SavedReaction) {
  if (input.kind === "remove") return;
  const saved = receipt.saved;
  if (!saved?.id || saved.actor !== scope.accountId || saved.reaction !== input.reaction)
    throw new Error("The saved reaction response did not match this request.");
  const matches =
    receipt.kind === "comment"
      ? receipt.saved?.comment === receipt.commentId && receipt.saved.project === scope.projectId
      : receipt.saved?.issue === scope.issueId;
  if (!matches) throw new Error("The saved reaction belonged to another work item.");
}
function replaceActorReaction<T extends { actor: string; reaction: string }>(
  rows: T[],
  actor: string | undefined,
  code: string,
  addition: T | undefined
) {
  const remaining = rows.filter((row) => row.actor !== actor || row.reaction !== code);
  return addition ? [...remaining, addition] : remaining;
}
function applyReaction(client: QueryClient, scope: DiscussionScope, input: ReactionChange, receipt: SavedReaction) {
  validateReaction(scope, input, receipt);
  if (receipt.kind === "comment") {
    client.setQueryData<TIssueComment[]>(discussionKey(scope, "comments"), (rows) =>
      rows?.map((row) =>
        row.id === receipt.commentId
          ? {
              ...row,
              comment_reactions: replaceActorReaction<TIssueCommentReaction>(
                row.comment_reactions,
                scope.accountId,
                input.reaction,
                receipt.saved
              ),
            }
          : row
      )
    );
  } else
    client.setQueryData<TIssueReaction[]>(discussionKey(scope, "reactions"), (rows = []) =>
      replaceActorReaction(rows, scope.accountId, input.reaction, receipt.saved)
    );
}
export function reactionMutationOptions(
  client: QueryClient,
  scope: DiscussionScope,
  collection: "comments" | "reactions"
) {
  const { captured, check, key, scopeId, reconcile } = mutationScope(client, scope, collection);
  return mutationOptions({
    mutationKey: [...key, "reaction-write"],
    scope: { id: scopeId },
    retry: false,
    mutationFn: async (input: ReactionWrite) => {
      check();
      if (collection === "comments") {
        if (!input.commentId) throw new Error("Choose a comment before reacting.");
        requireComment(client, scope, input.commentId);
      } else if (input.commentId) throw new Error("Unexpected comment reaction target.");
      await client.cancelQueries({ queryKey: key, exact: true });
      check();
      const change = resolveReaction(client, scope, input);
      const receipt = await sendReaction(scope, change);
      await reconcile(() => applyReaction(client, scope, change, receipt));
      return { captured };
    },
  });
}
