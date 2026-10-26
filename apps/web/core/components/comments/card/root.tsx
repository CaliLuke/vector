/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { CommentActions } from "@/components/comments/types";
import { useRef, useState } from "react";
// plane imports
import type { EditorRefApi } from "@plane/editor";
import type { TIssueComment } from "@plane/types";
// local imports
import { CommentQuickActions } from "../quick-actions";
import { CommentBlock } from "../comment-block";
import { CommentCardDisplay } from "./display";

type TCommentCard = {
  workspaceSlug: string;
  entityId: string;
  comment: TIssueComment | undefined;
  activityOperations: CommentActions;
  ends: "top" | "bottom" | undefined;
  showAccessSpecifier: boolean;
  showCopyLinkOption: boolean;
  disabled?: boolean;
  projectId?: string;
};

export const CommentCard = function CommentCard(props: TCommentCard) {
  const {
    workspaceSlug,
    entityId,
    comment,
    activityOperations,
    ends,
    showAccessSpecifier,
    showCopyLinkOption,
    disabled = false,
    projectId,
  } = props;
  // states
  const [isEditing, setIsEditing] = useState(false);
  // refs
  const readOnlyEditorRef = useRef<EditorRefApi>(null);
  // derived values
  const workspaceId = comment?.workspace;

  if (!comment || !workspaceId) return null;

  return (
    <CommentBlock comment={comment} ends={ends}>
      <CommentCardDisplay
        activityOperations={activityOperations}
        entityId={entityId}
        comment={comment}
        disabled={disabled}
        projectId={projectId}
        readOnlyEditorRef={readOnlyEditorRef}
        showAccessSpecifier={showAccessSpecifier}
        workspaceId={workspaceId}
        workspaceSlug={workspaceSlug}
        isEditing={isEditing}
        setIsEditing={setIsEditing}
        renderQuickActions={() => (
          <CommentQuickActions
            activityOperations={activityOperations}
            comment={comment}
            setEditMode={() => setIsEditing(true)}
            showAccessSpecifier={showAccessSpecifier}
            showCopyLinkOption={showCopyLinkOption}
          />
        )}
      />
    </CommentBlock>
  );
};
