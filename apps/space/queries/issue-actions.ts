/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { SitesIssueService } from "@plane/services";
import type { TIssuePublicComment } from "@plane/types";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { issueOptions } from "./issues";

const service = new SitesIssueService();
export function commentsOptions(anchor: string, issueId: string) {
  return queryOptions({
    queryKey: ["public", anchor, "issue", issueId, "comments"],
    queryFn: ({ signal }): Promise<TIssuePublicComment[]> => service.listComments(anchor, issueId, signal),
    staleTime: 30_000,
  });
}
export const issueActionKey = (anchor: string, issueId: string) => ["public", anchor, "issue", issueId, "mutation"];
const reportFailure = () => setToast({ type: TOAST_TYPE.ERROR, title: "Could not save", message: "Please try again." });
async function refreshIssue(client: QueryClient, anchor: string, issueId: string) {
  await Promise.all([
    client.invalidateQueries({ queryKey: ["public", anchor, "issues"] }),
    client.invalidateQueries({ queryKey: issueOptions(anchor, issueId).queryKey }),
  ]);
}
export function voteMutationOptions(client: QueryClient, anchor: string, issueId: string) {
  return mutationOptions({
    mutationKey: [...issueActionKey(anchor, issueId), "vote"],
    scope: { id: JSON.stringify([anchor, issueId, "vote"]) },
    mutationFn: (vote: 1 | -1 | null) =>
      vote === null ? service.removeVote(anchor, issueId) : service.addVote(anchor, issueId, { vote }),
    onSuccess: () => refreshIssue(client, anchor, issueId),
    onError: reportFailure,
  });
}
export function reactionMutationOptions(client: QueryClient, anchor: string, issueId: string) {
  return mutationOptions({
    mutationKey: [...issueActionKey(anchor, issueId), "reaction"],
    scope: { id: JSON.stringify([anchor, issueId, "reaction"]) },
    mutationFn: ({ reaction, remove }: { reaction: string; remove: boolean }) =>
      remove ? service.removeReaction(anchor, issueId, reaction) : service.addReaction(anchor, issueId, { reaction }),
    onSuccess: () => refreshIssue(client, anchor, issueId),
    onError: reportFailure,
  });
}
type CommentChange =
  | { kind: "add"; data: Partial<TIssuePublicComment> }
  | { kind: "update"; id: string; data: Partial<TIssuePublicComment> }
  | { kind: "delete"; id: string };
export function commentMutationOptions(client: QueryClient, anchor: string, issueId: string) {
  return mutationOptions({
    mutationKey: [...issueActionKey(anchor, issueId), "comment"],
    mutationFn: (change: CommentChange): Promise<TIssuePublicComment | undefined> => {
      switch (change.kind) {
        case "add":
          return service.addComment(anchor, issueId, change.data);
        case "update":
          return service.updateComment(anchor, issueId, change.id, change.data);
        case "delete":
          return service.removeComment(anchor, issueId, change.id);
      }
      throw new Error("Unknown comment operation");
    },
    onSuccess: () => client.invalidateQueries({ queryKey: commentsOptions(anchor, issueId).queryKey }),
    onError: reportFailure,
  });
}
export function commentReactionMutationOptions(
  client: QueryClient,
  anchor: string,
  issueId: string,
  commentId: string
) {
  return mutationOptions({
    mutationKey: [...issueActionKey(anchor, issueId), "comment-reaction", commentId],
    scope: { id: JSON.stringify([anchor, commentId, "reaction"]) },
    mutationFn: ({ reaction, remove }: { reaction: string; remove: boolean }) =>
      remove
        ? service.removeCommentReaction(anchor, commentId, reaction)
        : service.addCommentReaction(anchor, commentId, { reaction }),
    onSuccess: () => client.invalidateQueries({ queryKey: commentsOptions(anchor, issueId).queryKey }),
    onError: reportFailure,
  });
}
