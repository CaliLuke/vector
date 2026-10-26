/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import groupBy from "lodash-es/groupBy.js";
import { stringToEmoji } from "@plane/propel/emoji-icon-picker";
import type { EmojiReactionType } from "@plane/propel/emoji-reaction";
type Reaction = { id: string; reaction: string; actor: string; display_name?: string };

/** Derive the picker directly from the native canonical collection. */
export function buildReactionList(
  rows: readonly Reaction[],
  actorId: string | undefined,
  getMemberName: (id: string) => string | undefined
): EmojiReactionType[] {
  return Object.entries(groupBy(rows, "reaction")).map(([code, reactions]) => ({
    emoji: stringToEmoji(code),
    count: reactions.length,
    reacted: reactions.some((row) => row.actor === actorId),
    users: reactions
      .map((row) => getMemberName(row.actor) || row.display_name)
      .filter((name): name is string => !!name),
  }));
}
