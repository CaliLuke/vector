/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IPublicIssue } from "@plane/types";
import { useQuery } from "@tanstack/react-query";
import { publishSettingsOptions } from "@/queries/publishing";
import { useParams } from "react-router";
// plane utils
import { cn } from "@plane/utils/common";
// components
import { IssueEmojiReactions } from "@/components/issues/reactions/issue-emoji-reactions";
import { IssueVotes } from "@/components/issues/reactions/issue-vote-reactions";
// hooks

type Props = {
  issue: IPublicIssue;
};
export const BlockReactions = function BlockReactions(props: Props) {
  const { issue } = props;
  const { anchor } = useParams();
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));
  const { is_votes_enabled: canVote, is_reactions_enabled: canReact } = publishSettings ?? {};

  // if the user cannot vote or react then return empty
  if (!anchor || (!canVote && !canReact)) return null;

  return (
    <div className="flex w-full flex-wrap rounded-b-lg border-t-[1px] border-t-subtle-1 bg-surface-2 outline-transparent">
      <div className="flex flex-wrap items-center gap-2 px-3 py-2">
        {canVote && (
          <div
            className={cn(`flex items-center gap-2 pr-1`, {
              "after:ml-1 after:h-6 after:w-[1px] after:bg-layer-3": canReact,
            })}
          >
            <IssueVotes anchor={anchor} issue={issue} size="sm" />
          </div>
        )}
        {canReact && (
          <div className="flex flex-wrap items-center gap-2">
            <IssueEmojiReactions anchor={anchor} issue={issue} />
          </div>
        )}
      </div>
    </div>
  );
};
