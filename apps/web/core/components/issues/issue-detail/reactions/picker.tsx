/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { cn } from "@plane/utils/common";
import { EmojiReactionGroup, EmojiReactionPicker } from "@plane/propel/emoji-reaction";
import type { EmojiReactionType } from "@plane/propel/emoji-reaction";

type ReactionPickerProps = {
  reactions: EmojiReactionType[];
  onReact: (reaction: string) => void;
  disabled: boolean;
  className?: string;
};

export function ReactionPicker({ reactions, onReact, disabled, className }: ReactionPickerProps) {
  const [isPickerOpen, setIsPickerOpen] = useState(false);

  const handleReactionClick = (emoji: string) => {
    if (disabled) return;
    const codePoints = Array.from(emoji).map((character) => character.codePointAt(0));
    onReact(codePoints.join("-"));
  };

  return (
    <div className={cn("relative mt-4", className)}>
      <EmojiReactionPicker
        isOpen={isPickerOpen}
        handleToggle={setIsPickerOpen}
        onChange={onReact}
        disabled={disabled}
        label={
          <EmojiReactionGroup
            reactions={reactions}
            onReactionClick={handleReactionClick}
            showAddButton={!disabled}
            onAddReaction={() => setIsPickerOpen(true)}
          />
        }
        placement="bottom-start"
      />
    </div>
  );
}
