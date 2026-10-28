/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useQuery } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/session";
import { publishSettingsOptions } from "@/queries/publishing";
import { Link, useLocation } from "react-router";

import { Lock } from "lucide-react";
// plane imports
import { Button } from "@plane/propel/button";
// components
import { AddComment } from "@/components/issues/peek-overview/comment/add-comment";
import { CommentCard } from "@/components/issues/peek-overview/comment/comment-detail-card";
// hooks
import { commentsOptions } from "@/queries/issue-actions";
import useIsInIframe from "@/hooks/use-is-in-iframe";
// types

import type { IPublicIssue } from "@plane/types";

type Props = {
  anchor: string;
  issueDetails: IPublicIssue;
};

export const PeekOverviewIssueActivity = function PeekOverviewIssueActivity(props: Props) {
  const { anchor, issueDetails } = props;
  // router
  const { pathname } = useLocation();
  const commentsQuery = useRecordCollection(commentsOptions(anchor, issueDetails.id));
  const { data: currentUser } = useQuery(currentUserOptions);
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));
  const { is_comments_enabled: canComment } = publishSettings ?? {};
  // derived values
  const comments = commentsQuery.data ?? [];
  const isInIframe = useIsInIframe();

  return (
    <div className="pb-10">
      <h4 className="font-medium">Comments</h4>
      <div className="mt-4">
        <div className="space-y-4">
          {commentsQuery.isError && (
            <div role="alert">
              Could not load comments. <button onClick={() => void commentsQuery.refetch()}>Retry</button>
            </div>
          )}
          {comments.map((comment) => (
            <CommentCard issueId={issueDetails.id} key={comment.id} anchor={anchor} comment={comment} />
          ))}
        </div>
        {!isInIframe &&
          (currentUser ? (
            <>
              {canComment && (
                <div className="mt-4">
                  <AddComment key={issueDetails.id} issueId={issueDetails.id} anchor={anchor} disabled={!currentUser} />
                </div>
              )}
            </>
          ) : (
            <div className="mt-4 flex items-center justify-between gap-2 rounded-sm border border-strong bg-layer-2 px-2 py-2.5">
              <p className="flex items-center gap-2 overflow-hidden text-13 break-words text-secondary">
                <Lock className="size-3 shrink-0" />
                Sign in to add your comment
              </p>
              <Link to={`/?next_path=${pathname}`}>
                <Button variant="primary">Sign in</Button>
              </Link>
            </div>
          ))}
      </div>
    </div>
  );
};
