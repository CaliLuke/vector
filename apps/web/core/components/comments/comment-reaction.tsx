/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCurrentUserId } from "@/hooks/use-current-user";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { buildReactionList } from "@/components/issues/issue-detail/reactions/data";
import type { CommentActions } from "@/components/comments/types";
import { useState } from "react";
// plane imports
import { EmojiReactionGroup, EmojiReactionPicker } from "@plane/propel/emoji-reaction";
import type { TIssueComment } from "@plane/types";
// helpers
// local imports

export type TProps = {
  comment: TIssueComment;
  disabled?: boolean;
  activityOperations: CommentActions;
};

export const CommentReactions = function CommentReactions(props: TProps) {
  const { comment, activityOperations, disabled = false } = props;
  // state
  const [isPickerOpen, setIsPickerOpen] = useState(false);

  const accountId = useCurrentUserId();
  const { membersById } = useWorkspaceMemberRoster(comment.workspace_detail?.slug);
  const reactions = buildReactionList(
    comment.comment_reactions,
    accountId,
    (id) => membersById?.get(id)?.member.display_name
  );

  const handleReactionClick = (emoji: string) => {
    if (disabled || !activityOperations.isVerified()) return;
    // Convert emoji back to decimal string format for the API
    const emojiCodePoints = Array.from(emoji).map((char) => char.codePointAt(0));
    const reactionString = emojiCodePoints.join("-");
    void activityOperations.react(comment.id, reactionString).catch(() => {});
  };

  const handleEmojiSelect = (emoji: string) => {
    if (!activityOperations.isVerified()) return;
    // emoji is already in decimal string format from EmojiReactionPicker
    void activityOperations.react(comment.id, emoji).catch(() => {});
  };

  // Don't render anything if there are no reactions and it's disabled
  if (reactions.length === 0 && disabled) return null;

  // Don't show the add button if there are no reactions
  const showAddButton = !disabled && reactions.length > 0;

  return (
    <div className="relative">
      <EmojiReactionPicker
        isOpen={isPickerOpen}
        handleToggle={setIsPickerOpen}
        onChange={handleEmojiSelect}
        disabled={disabled}
        label={
          <EmojiReactionGroup
            reactions={reactions}
            onReactionClick={handleReactionClick}
            showAddButton={showAddButton}
            onAddReaction={() => setIsPickerOpen(true)}
          />
        }
        placement="bottom-start"
      />
    </div>
  );
};
