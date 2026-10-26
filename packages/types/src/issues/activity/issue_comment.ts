/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssueCommentReaction } from "./issue_comment_reaction";
import type { JSONContent } from "../../editor";
import type { EIssueCommentAccessSpecifier } from "../../enums";
import type { IWorkspaceLite } from "../../workspace";
import type {
  TIssueActivityWorkspaceDetail,
  TIssueActivityProjectDetail,
  TIssueActivityIssueDetail,
  TIssueActivityUserDetail,
} from "./base";

export type TIssueComment = {
  id: string;
  workspace: string;
  workspace_detail: TIssueActivityWorkspaceDetail;
  project: string;
  project_detail: TIssueActivityProjectDetail;
  issue: string;
  issue_detail: TIssueActivityIssueDetail;
  actor: string;
  actor_detail: TIssueActivityUserDetail;
  created_at: string;
  edited_at?: string | undefined;
  updated_at: string;
  created_by: string | undefined;
  updated_by: string | undefined;
  attachments: any[];
  comment_reactions: TIssueCommentReaction[];
  parent?: string | null;
  comment_stripped: string;
  comment_html: string;
  comment_json: JSONContent;
  external_id: string | undefined;
  external_source: string | undefined;
  access: EIssueCommentAccessSpecifier;
};

export interface ActorDetail {
  avatar_url?: string;
  display_name?: string;
  first_name?: string;
  is_bot?: boolean;
  id?: string;
  last_name?: string;
}

export interface IssueDetail {
  id: string;
  name: string;
  description: Description;
  description_html: string;
  priority: string;
  start_date: null;
  target_date: null;
  sequence_id: number;
  sort_order: number;
}

export interface Description {
  type: string;
  content: DescriptionContent[];
}

export interface DescriptionContent {
  type: string;
  attrs?: Attrs;
  content: ContentContent[];
}

export interface Attrs {
  level: number;
}

export interface ContentContent {
  text: string;
  type: string;
}

export interface ProjectDetail {
  id: string;
  identifier: string;
  name: string;
  cover_image: string;
  icon_prop: null;
  emoji: string;
  description: string;
}

export type TIssuePublicComment = {
  actor_detail: ActorDetail;
  access: string;
  actor: string;
  attachments: any[];
  comment_html: string;
  comment_reactions: {
    actor_detail: ActorDetail;
    comment: string;
    id: string;
    reaction: string;
  }[];
  comment_stripped: string;
  created_at: Date;
  created_by: string;
  id: string;
  is_member: boolean;
  issue: string;
  issue_detail: IssueDetail;
  project: string;
  project_detail: ProjectDetail;
  updated_at: Date;
  updated_by: string;
  workspace: string;
  workspace_detail: IWorkspaceLite;
};
