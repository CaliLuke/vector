/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IUserLite } from "../users";

export type TIssueReaction = {
  actor: string;
  id: string;
  issue: string;
  reaction: string;
  display_name: string;
};

export interface IIssuePublicReaction {
  actor_details: IUserLite;
  reaction: string;
}

export interface IPublicVote {
  vote: -1 | 1;
  actor_details: IUserLite;
}
