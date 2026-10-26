/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery, useMutation, useQueryClient, useIsMutating } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/session";
import { ArrowDown, ArrowUp } from "lucide-react";

import { useLocation, useNavigate } from "react-router";
// plane imports
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
// helpers
// hooks

import type { IPublicIssue } from "@plane/types";
import { voteMutationOptions, issueActionKey } from "@/queries/issue-actions";
import useIsInIframe from "@/hooks/use-is-in-iframe";

type TIssueVotes = {
  anchor: string;
  issue: IPublicIssue;
  size?: "md" | "sm";
};

export const IssueVotes = function IssueVotes(props: TIssueVotes) {
  const { anchor, issue, size = "md" } = props;
  // states
  // router
  const navigate = useNavigate();
  const location = useLocation();
  const signInPath = `/?${new URLSearchParams({ next_path: location.pathname + location.search })}`;
  const client = useQueryClient();
  const { data: user } = useQuery(currentUserOptions);

  const isInIframe = useIsInIframe();

  const issueId = issue.id;

  const votes = issue.vote_items ?? [];

  const allUpVotes = votes.filter((vote) => vote.vote === 1);
  const allDownVotes = votes.filter((vote) => vote.vote === -1);

  const isUpVotedByUser = allUpVotes.some((vote) => vote.actor_details?.id === user?.id);
  const isDownVotedByUser = allDownVotes.some((vote) => vote.actor_details?.id === user?.id);

  const mutation = useMutation(voteMutationOptions(client, anchor, issueId));
  const isSubmitting = useIsMutating({ mutationKey: [...issueActionKey(anchor, issueId), "vote"] }) > 0;
  const handleVote = (value: 1 | -1) => {
    if (isSubmitting) return;
    const existing = votes.some((vote) => vote.actor_details?.id === user?.id && vote.vote === value);
    mutation.mutate(existing ? null : value);
  };

  const VOTES_LIMIT = 1000;

  // derived values
  const votingDimensions = size === "sm" ? "px-1 h-6 min-w-9" : "px-2 h-7";

  return (
    <div className="flex items-center gap-2">
      {/* upvote button 👇 */}
      <Tooltip
        tooltipContent={
          <div>
            {allUpVotes.length > 0 ? (
              <>
                {allUpVotes
                  .map((r) => r.actor_details?.display_name)
                  .splice(0, VOTES_LIMIT)
                  .join(", ")}
                {allUpVotes.length > VOTES_LIMIT && " and " + (allUpVotes.length - VOTES_LIMIT) + " more"}
              </>
            ) : (
              "No upvotes yet"
            )}
          </div>
        }
      >
        <button
          type="button"
          disabled={isSubmitting}
          onClick={() => {
            if (isInIframe) return;
            if (user) handleVote(1);
            else void navigate(signInPath);
          }}
          className={cn(
            "flex items-center justify-center gap-x-1 overflow-hidden rounded-sm border hover:bg-layer-transparent-hover focus:outline-none",
            votingDimensions,
            {
              "border-accent-strong-200 text-accent-secondary": isUpVotedByUser,
              "border-strong": !isUpVotedByUser,
              "cursor-default": isInIframe,
            }
          )}
        >
          <ArrowUp className="size-3.5 shrink-0" />
          <span className="text-13 font-regular transition-opacity ease-in-out">{allUpVotes.length}</span>
        </button>
      </Tooltip>

      {/* downvote button 👇 */}
      <Tooltip
        tooltipContent={
          <div>
            {allDownVotes.length > 0 ? (
              <>
                {allDownVotes
                  .map((r) => r.actor_details.display_name)
                  .splice(0, VOTES_LIMIT)
                  .join(", ")}
                {allDownVotes.length > VOTES_LIMIT && " and " + (allDownVotes.length - VOTES_LIMIT) + " more"}
              </>
            ) : (
              "No downvotes yet"
            )}
          </div>
        }
      >
        <button
          type="button"
          disabled={isSubmitting}
          onClick={() => {
            if (isInIframe) return;
            if (user) handleVote(-1);
            else void navigate(signInPath);
          }}
          className={cn(
            "flex items-center justify-center gap-x-1 overflow-hidden rounded-sm border hover:bg-layer-transparent-hover focus:outline-none",
            votingDimensions,
            {
              "border-danger-strong text-danger-primary": isDownVotedByUser,
              "border-strong": !isDownVotedByUser,
              "cursor-default": isInIframe,
            }
          )}
        >
          <ArrowDown className="size-3.5 shrink-0" />
          <span className="text-13 font-regular transition-opacity ease-in-out">{allDownVotes.length}</span>
        </button>
      </Tooltip>
    </div>
  );
};
