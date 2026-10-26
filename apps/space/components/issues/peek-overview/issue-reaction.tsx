/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IPublicIssue } from "@plane/types";
import { useQuery } from "@tanstack/react-query";
import { publishSettingsOptions } from "@/queries/publishing";
// components
import { IssueEmojiReactions } from "@/components/issues/reactions/issue-emoji-reactions";
import { IssueVotes } from "@/components/issues/reactions/issue-vote-reactions";
// hooks
import useIsInIframe from "@/hooks/use-is-in-iframe";

type Props = {
  anchor: string;
  issue: IPublicIssue;
};

export const IssueReactions = function IssueReactions(props: Props) {
  const { anchor, issue } = props;
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));
  const { is_votes_enabled: canVote, is_reactions_enabled: canReact } = publishSettings ?? {};
  const isInIframe = useIsInIframe();

  return (
    <div className="mt-4 flex items-center gap-3">
      {canVote && (
        <div className="flex items-center gap-2">
          <IssueVotes issue={issue} anchor={anchor} />
        </div>
      )}
      {!isInIframe && canReact && (
        <div className="flex items-center gap-2">
          <IssueEmojiReactions issue={issue} anchor={anchor} />
        </div>
      )}
    </div>
  );
};
