/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient, useIsMutating } from "@tanstack/react-query";
import type { IPublicIssue } from "@plane/types";
import { issueActionKey, reactionMutationOptions } from "@/queries/issue-actions";
import { EmojiReactions } from "./emoji-reactions";

export function IssueEmojiReactions({ anchor, issue }: { anchor: string; issue: IPublicIssue }) {
  const mutation = useMutation(reactionMutationOptions(useQueryClient(), anchor, issue.id));
  const pending = useIsMutating({ mutationKey: [...issueActionKey(anchor, issue.id), "reaction"] }) > 0;
  const reactions = (issue.reaction_items ?? []).map(({ reaction, actor_details }) => ({
    reaction,
    actor: actor_details,
  }));
  return <EmojiReactions reactions={reactions} pending={pending} onToggle={(change) => mutation.mutate(change)} />;
}
