/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { CommentActions } from "@/components/comments/types";
import { useLocation } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { useState } from "react";

// plane imports
import type { EditorRefApi } from "@plane/editor";
import { GlobeIcon, LockIcon } from "@plane/propel/icons";
import { EIssueCommentAccessSpecifier } from "@plane/types";
import type { IWorkspaceMember, TIssueComment } from "@plane/types";
import { calculateTimeAgo, renderFormattedDate, renderFormattedTime } from "@plane/utils/datetime";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
// components
import { LiteTextEditor } from "@/components/editor/lite-text";
// local imports
import { CommentReactions } from "../comment-reaction";
import { CommentCardEditForm } from "./edit-form";
import { EmojiReactionButton, EmojiReactionPicker } from "@plane/propel/emoji-reaction";
import { Avatar } from "@plane/ui/avatar";
import { Tooltip } from "@plane/ui/tooltip";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useCommentHighlight } from "./use-comment-highlight";

export type TCommentCardDisplayProps = {
  activityOperations: CommentActions;
  comment: TIssueComment;
  disabled: boolean;
  entityId: string;
  projectId?: string;
  readOnlyEditorRef: React.RefObject<EditorRefApi | null>;
  showAccessSpecifier: boolean;
  workspaceId: string;
  workspaceSlug: string;
  isEditing?: boolean;
  setIsEditing?: (isEditing: boolean) => void;
  renderFooter?: (ReactionsComponent: ReactNode | null) => ReactNode;
  renderQuickActions?: () => ReactNode;
};

type CommentHeaderProps = Pick<
  TCommentCardDisplayProps,
  "activityOperations" | "comment" | "disabled" | "showAccessSpecifier" | "renderQuickActions"
> & {
  displayName: string | undefined;
  avatarUrl: string | undefined;
};

function commentAuthor(comment: TIssueComment, member: IWorkspaceMember["member"] | undefined) {
  const actor = comment.actor_detail;
  return {
    displayName: actor?.is_bot ? `${actor.first_name}Bot` : (member?.display_name ?? actor?.display_name),
    avatarUrl: member?.avatar_url ?? actor?.avatar_url,
  };
}

const CommentCardHeader = function CommentCardHeader(props: CommentHeaderProps) {
  const { activityOperations, comment, disabled, showAccessSpecifier, renderQuickActions, displayName, avatarUrl } =
    props;
  const [isPickerOpen, setIsPickerOpen] = useState(false);
  const handleEmojiSelect = (emoji: string) => {
    if (!activityOperations.isVerified()) return;
    void activityOperations.react(comment.id, emoji).catch(() => {});
  };

  return (
    <>
      {showAccessSpecifier && (
        <div className="absolute top-2.5 right-2.5 z-[1] text-tertiary">
          {comment.access === EIssueCommentAccessSpecifier.INTERNAL ? (
            <LockIcon className="size-3" />
          ) : (
            <GlobeIcon className="size-3" />
          )}
        </div>
      )}
      <div className="relative mb-3 flex w-full items-center gap-2">
        <Avatar size="sm" name={displayName} src={getFileURL(avatarUrl ?? "")} className="shrink-0" />
        <div className="flex flex-1 flex-wrap items-center gap-1">
          <div className="text-caption-sm-medium">{displayName}</div>
          <div className="text-caption-sm-regular text-tertiary">
            commented{" "}
            <Tooltip
              tooltipContent={`${renderFormattedDate(comment.created_at)} at ${renderFormattedTime(comment.created_at)}`}
              position="bottom"
            >
              <span className="text-tertiary">
                {calculateTimeAgo(comment.created_at)}
                {comment.edited_at && " (edited)"}
              </span>
            </Tooltip>
          </div>
        </div>
        {!disabled && (
          <div className="flex shrink-0 items-center gap-1">
            <EmojiReactionPicker
              isOpen={isPickerOpen}
              handleToggle={setIsPickerOpen}
              onChange={handleEmojiSelect}
              disabled={disabled}
              label={<EmojiReactionButton onAddReaction={() => setIsPickerOpen(true)} />}
              placement="bottom-start"
            />
            {renderQuickActions ? renderQuickActions() : null}
          </div>
        )}
      </div>
    </>
  );
};

type CommentBodyProps = Pick<
  TCommentCardDisplayProps,
  | "activityOperations"
  | "comment"
  | "disabled"
  | "projectId"
  | "readOnlyEditorRef"
  | "workspaceId"
  | "workspaceSlug"
  | "isEditing"
  | "setIsEditing"
  | "renderFooter"
> & { highlightClassName: string };

const CommentCardBody = function CommentCardBody(props: CommentBodyProps) {
  const {
    activityOperations,
    comment,
    disabled,
    projectId,
    readOnlyEditorRef,
    workspaceId,
    workspaceSlug,
    isEditing,
    setIsEditing,
    renderFooter,
    highlightClassName,
  } = props;
  const hasReactions = comment.comment_reactions.length > 0;
  const shouldRenderReactions = hasReactions && !disabled;

  if (isEditing && setIsEditing) {
    return (
      <CommentCardEditForm
        activityOperations={activityOperations}
        comment={comment}
        readOnlyEditorRef={readOnlyEditorRef}
        setIsEditing={setIsEditing}
        projectId={projectId}
        workspaceId={workspaceId}
        workspaceSlug={workspaceSlug}
      />
    );
  }

  return (
    <>
      <LiteTextEditor
        editable={false}
        ref={readOnlyEditorRef}
        id={comment.id}
        initialValue={comment.comment_html ?? ""}
        workspaceId={workspaceId}
        workspaceSlug={workspaceSlug}
        containerClassName={cn("!py-1 transition-[border-color] duration-500", highlightClassName)}
        projectId={projectId?.toString()}
        displayConfig={{ fontSize: "small-font" }}
        parentClassName="border-none"
      />
      {shouldRenderReactions &&
        (renderFooter ? (
          renderFooter(
            <CommentReactions comment={comment} disabled={disabled} activityOperations={activityOperations} />
          )
        ) : (
          <CommentReactions comment={comment} disabled={disabled} activityOperations={activityOperations} />
        ))}
    </>
  );
};

export const CommentCardDisplay = function CommentCardDisplay(props: TCommentCardDisplayProps) {
  const {
    activityOperations,
    comment,
    disabled,
    projectId,
    readOnlyEditorRef,
    showAccessSpecifier,
    workspaceId,
    workspaceSlug,
    isEditing = false,
    setIsEditing,
    renderFooter,
    renderQuickActions,
  } = props;
  // store hooks
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  // derived values
  const { displayName, avatarUrl } = commentAuthor(comment, memberById(comment.actor)?.member);

  // navigation
  const pathname = useLocation({ select: (location) => location.pathname });
  // derived values
  const commentBlockId = `comment-${comment.id}`;
  const highlightClassName = useCommentHighlight(commentBlockId, pathname);

  return (
    <div id={commentBlockId} className="relative flex flex-col gap-2">
      <CommentCardHeader
        activityOperations={activityOperations}
        comment={comment}
        disabled={disabled}
        showAccessSpecifier={showAccessSpecifier}
        displayName={displayName}
        avatarUrl={avatarUrl}
        renderQuickActions={renderQuickActions}
      />
      <CommentCardBody
        activityOperations={activityOperations}
        comment={comment}
        disabled={disabled}
        projectId={projectId}
        readOnlyEditorRef={readOnlyEditorRef}
        workspaceId={workspaceId}
        workspaceSlug={workspaceSlug}
        isEditing={isEditing}
        setIsEditing={setIsEditing}
        renderFooter={renderFooter}
        highlightClassName={highlightClassName}
      />
    </div>
  );
};
