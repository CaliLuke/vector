/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { CommentActions } from "@/components/comments/types";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import type { RefObject } from "react";
import { useEffect, useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { EFileAssetType } from "@plane/types";
import type { EditorRefApi } from "@plane/editor";
import { CheckIcon, CloseIcon } from "@plane/propel/icons";
// plane imports
import type { TIssueComment } from "@plane/types";
import { cn } from "@plane/utils/common";
import { isCommentEmpty } from "@plane/utils/string";
import { FormErrors } from "@plane/ui";
// components
import { LiteTextEditor } from "@/components/editor/lite-text";

type Props = {
  activityOperations: CommentActions;
  comment: TIssueComment;
  projectId?: string;
  readOnlyEditorRef: RefObject<EditorRefApi | null>;
  setIsEditing: (isEditing: boolean) => void;
  workspaceId: string;
  workspaceSlug: string;
};

export function CommentCardEditForm(props: Props) {
  const { activityOperations, comment, projectId, readOnlyEditorRef, setIsEditing, workspaceId, workspaceSlug } = props;
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const current = useCommittedScope(JSON.stringify([workspaceSlug, projectId, comment.id]));
  const form = useForm({
    defaultValues: { comment_html: comment.comment_html ?? "" },
    onSubmit: async ({ value }) => {
      if (isCommentEmpty(value.comment_html) || !editorRef.current?.isEditorReadyToDiscard())
        throw new Error("Enter a comment and wait for the editor before saving.");
      await activityOperations.updateComment(comment.id, value);
      if (!current() || !activityOperations.isCurrent()) return;
      readOnlyEditorRef.current?.setEditorValue(value.comment_html);
      setIsEditing(false);
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const commentHTML = useStore(form.store, (state) => state.values.comment_html);
  const isEmpty = isCommentEmpty(commentHTML);
  useEffect(() => {
    editorRef.current?.focus("end");
  }, []);

  return (
    <form
      className="flex flex-col gap-2"
      onSubmit={(event) => {
        event.preventDefault();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => {});
      }}
    >
      <div inert={isSubmitting}>
        <form.Field
          name="comment_html"
          validators={{
            onSubmit: ({ value }) => {
              if (isCommentEmpty(value)) return "Enter a comment before saving.";
              return editorRef.current?.isEditorReadyToDiscard() ? undefined : "Comment editor is still loading.";
            },
          }}
        >
          {(field) => (
            <>
              <LiteTextEditor
                editable
                workspaceId={workspaceId}
                workspaceSlug={workspaceSlug}
                ref={editorRef}
                id={comment.id}
                initialValue={field.state.value ?? ""}
                value={null}
                onChange={(_comment_json, comment_html) => field.handleChange(comment_html)}
                onEnterKeyPress={(e) => {
                  e?.preventDefault();
                  form.setErrorMap({ onSubmit: undefined });
                  void form.handleSubmit().catch(() => {});
                }}
                showSubmitButton={false}
                fileAsset={{ entity_identifier: comment.id, entity_type: EFileAssetType.COMMENT_DESCRIPTION }}
                projectId={projectId}
                parentClassName="p-2 bg-surface-1"
                displayConfig={{ fontSize: "small-font" }}
              />
              <FormErrors errors={field.state.meta.errors} />
            </>
          )}
        </form.Field>
      </div>
      <div className="flex gap-2 self-end">
        {!isEmpty && (
          <button
            type="submit"
            aria-label="Save comment"
            disabled={isSubmitting}
            className={cn(
              "group grid size-7 place-items-center rounded-lg border border-success-subtle bg-success-subtle shadow-raised-100 duration-300",
              isSubmitting ? "" : "hover:bg-success-subtle-1"
            )}
          >
            <CheckIcon className="size-4 text-success-primary" />
          </button>
        )}
        <button
          type="button"
          disabled={isSubmitting}
          aria-label="Cancel editing"
          className={cn(
            "group grid size-7 place-items-center rounded-lg border border-danger-subtle bg-danger-subtle shadow-raised-100 duration-300",
            isSubmitting ? "" : "hover:bg-danger-subtle-hover"
          )}
          onClick={() => {
            if (current()) setIsEditing(false);
            editorRef.current?.setEditorValue(comment.comment_html ?? "<p></p>");
          }}
        >
          <CloseIcon className="size-4 text-danger-primary" />
        </button>
      </div>
    </form>
  );
}
