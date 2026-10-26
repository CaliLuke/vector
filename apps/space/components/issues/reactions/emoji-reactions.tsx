/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";
import { stringToEmoji } from "@plane/propel/emoji-icon-picker";
import { EmojiReactionGroup, EmojiReactionPicker } from "@plane/propel/emoji-reaction";
import type { ActorDetail } from "@plane/types";
import { currentUserOptions } from "@/queries/session";
import useIsInIframe from "@/hooks/use-is-in-iframe";

type Reaction = { reaction: string; actor: ActorDetail };
type Props = {
  reactions: Reaction[];
  pending: boolean;
  onToggle: (change: { reaction: string; remove: boolean }) => void;
};
export function EmojiReactions({ reactions, pending, onToggle }: Props) {
  const [open, setOpen] = useState(false);
  const { data: user } = useQuery(currentUserOptions);
  const location = useLocation();
  const navigate = useNavigate();
  const embedded = useIsInIframe();
  const groups = new Map<string, Reaction[]>();
  for (const item of reactions) groups.set(item.reaction, [...(groups.get(item.reaction) ?? []), item]);
  const displayed = Array.from(groups, ([reaction, items]) => ({
    emoji: stringToEmoji(reaction),
    count: items.length,
    reacted: items.some((item) => item.actor.id === user?.id),
    users: items
      .map((item) => item.actor.display_name)
      .filter((name): name is string => !!name)
      .slice(0, 1000),
  }));
  const toggle = (reaction: string) => {
    if (pending || embedded) return;
    if (!user) {
      void navigate(`/?${new URLSearchParams({ next_path: location.pathname + location.search })}`);
      return;
    }
    onToggle({ reaction, remove: reactions.some((item) => item.reaction === reaction && item.actor.id === user.id) });
  };
  return (
    <EmojiReactionPicker
      isOpen={open}
      handleToggle={setOpen}
      onChange={toggle}
      disabled={pending || embedded}
      label={
        <EmojiReactionGroup
          reactions={displayed}
          onReactionClick={(emoji) => toggle(Array.from(emoji, (character) => character.codePointAt(0)).join("-"))}
          showAddButton={!embedded}
          onAddReaction={() => setOpen(true)}
        />
      }
      placement="bottom-start"
    />
  );
}
