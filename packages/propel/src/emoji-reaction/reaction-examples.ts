/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useState } from "react";
import type { EmojiReactionType } from "./emoji-reaction";

export function toggleExampleReaction(reactions: EmojiReactionType[], emoji: string): EmojiReactionType[] {
  return reactions.map((reaction) =>
    reaction.emoji === emoji
      ? { ...reaction, reacted: !reaction.reacted, count: reaction.count + (reaction.reacted ? -1 : 1) }
      : reaction
  );
}

export function useExampleReactions(initial: EmojiReactionType[]) {
  const [reactions, setReactions] = useState(initial);
  const addReaction = (emoji: string) =>
    setReactions((previous) =>
      previous.some((reaction) => reaction.emoji === emoji)
        ? previous.map((reaction) =>
            reaction.emoji === emoji ? { ...reaction, count: reaction.count + 1, reacted: true } : reaction
          )
        : [...previous, { emoji, count: 1, reacted: true, users: ["You"] }]
    );
  const toggleReaction = (emoji: string) => setReactions((previous) => toggleExampleReaction(previous, emoji));
  return { reactions, addReaction, toggleReaction };
}
