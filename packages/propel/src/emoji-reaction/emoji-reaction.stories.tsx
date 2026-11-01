/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useExampleReactions } from "./reaction-examples";
import { ReactionMessageExample } from "./message-example";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { EmojiReaction, EmojiReactionGroup, EmojiReactionButton } from "./emoji-reaction";

const meta = {
  title: "Components/Emoji/EmojiReaction",
  component: EmojiReaction,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof EmojiReaction>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Single: Story = {
  args: {
    emoji: "👍",
    count: 5,
    reacted: false,
    users: ["Alice", "Bob", "Charlie"],
  },
};

export const Reacted: Story = {
  args: {
    emoji: "❤️",
    count: 12,
    reacted: true,
    users: ["Alice", "Bob", "Charlie", "David", "Emma", "Frank"],
  },
};

export const Interactive: Story = {
  args: {
    emoji: "👍",
    count: 0,
  },
  render() {
    const [reacted, setReacted] = useState(false);
    const [count, setCount] = useState(5);

    const handleClick = () => {
      setReacted(!reacted);
      setCount(reacted ? count - 1 : count + 1);
    };

    return (
      <div className="flex flex-col items-center gap-4">
        <EmojiReaction
          emoji="👍"
          count={count}
          reacted={reacted}
          users={["Alice", "Bob", "Charlie"]}
          onReactionClick={handleClick}
        />
        <p className="text-13 text-placeholder">Click to toggle reaction</p>
      </div>
    );
  },
};

export const WithTooltip: Story = {
  args: {
    emoji: "🎉",
    count: 8,
    reacted: false,
    users: ["Alice", "Bob", "Charlie", "David", "Emma", "Frank", "Grace", "Henry"],
  },
};

export const WithoutCount: Story = {
  args: {
    emoji: "🔥",
    count: 0,
    reacted: false,
    showCount: false,
  },
};

const initialReactions = [
  { emoji: "👍", count: 5, reacted: false, users: ["Alice", "Bob", "Charlie"] },
  { emoji: "❤️", count: 12, reacted: true, users: ["David", "Emma", "Frank"] },
  { emoji: "🎉", count: 3, reacted: false, users: ["Grace"] },
];

export const MultipleReactions: Story = {
  args: {
    emoji: "👍",
    count: 0,
  },
  render() {
    const { reactions, toggleReaction: handleReactionClick } = useExampleReactions(initialReactions);

    return (
      <div className="flex gap-2">
        {reactions.map((reaction) => (
          <EmojiReaction
            key={reaction.emoji}
            emoji={reaction.emoji}
            count={reaction.count}
            reacted={reaction.reacted}
            users={reaction.users}
            onReactionClick={handleReactionClick}
          />
        ))}
      </div>
    );
  },
};

const handleAdd = () => {
  alert("Add reaction clicked");
};

export const AddButton: Story = {
  args: {
    emoji: "➕",
    count: 0,
  },
  render() {
    return <EmojiReactionButton onAddReaction={handleAdd} />;
  },
};

export const ReactionGroup: Story = {
  args: {
    emoji: "👍",
    count: 0,
  },
  render() {
    const { reactions, toggleReaction: handleReactionClick } = useExampleReactions([
      ...initialReactions,
      { emoji: "🔥", count: 8, reacted: false, users: ["Henry", "Ivy"] },
    ]);

    return <EmojiReactionGroup reactions={reactions} onReactionClick={handleReactionClick} onAddReaction={handleAdd} />;
  },
};

export const InMessageContext: Story = {
  args: {
    emoji: "👍",
    count: 0,
  },
  render() {
    const { reactions, toggleReaction: handleReactionClick } = useExampleReactions([
      { emoji: "👍", count: 5, reacted: false, users: ["Alice", "Bob", "Charlie"] },
      { emoji: "❤️", count: 2, reacted: true, users: ["You", "David"] },
    ]);

    return (
      <ReactionMessageExample message="Hey everyone! Just wanted to share some exciting news about our project launch next week!">
        <EmojiReactionGroup reactions={reactions} onReactionClick={handleReactionClick} />
      </ReactionMessageExample>
    );
  },
};

export const ManyUsers: Story = {
  args: {
    emoji: "🎉",
    count: 47,
    reacted: true,
    users: ["Alice", "Bob", "Charlie", "David", "Emma", "Frank", "Grace", "Henry", "Ivy", "Jack", "Kate", "Liam"],
  },
};
