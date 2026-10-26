/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssueComment } from "@plane/types";

export type CommentActions = {
  /** Whether this editor may apply completion feedback to its captured discussion. */
  isCurrent: () => boolean;
  /** New asset/status writes require verified identity and permission sources. */
  isVerified: () => boolean;
  copyCommentLink: (commentId: string) => void;
  createComment: (data: Partial<TIssueComment>) => Promise<Partial<TIssueComment> | undefined>;
  updateComment: (commentId: string, data: Partial<TIssueComment>) => Promise<void>;
  removeComment: (commentId: string) => Promise<void>;
  react: (commentId: string, reactionEmoji: string) => Promise<void>;
};
