/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient, useIsMutating } from "@tanstack/react-query";
import type { TIssuePublicComment } from "@plane/types";
import { commentReactionMutationOptions, issueActionKey } from "@/queries/issue-actions";
import { EmojiReactions } from "@/components/issues/reactions/emoji-reactions";

type Props = { anchor: string; issueId: string; comment: TIssuePublicComment };
export function CommentReactions({ anchor, issueId, comment }: Props) {
  const mutation = useMutation(commentReactionMutationOptions(useQueryClient(), anchor, issueId, comment.id));
  const pending =
    useIsMutating({ mutationKey: [...issueActionKey(anchor, issueId), "comment-reaction", comment.id] }) > 0;
  const reactions = (comment.comment_reactions ?? []).map(({ reaction, actor_detail }) => ({
    reaction,
    actor: actor_detail,
  }));
  return <EmojiReactions reactions={reactions} pending={pending} onToggle={(change) => mutation.mutate(change)} />;
}
