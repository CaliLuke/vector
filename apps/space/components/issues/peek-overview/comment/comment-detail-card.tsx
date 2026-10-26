/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { SitesFileService } from "@plane/services";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/session";
import { publishSettingsOptions } from "@/queries/publishing";
import React, { useRef, useState } from "react";

import { useForm, useStore } from "@tanstack/react-form";
import { MessageSquare, MoreVertical } from "lucide-react";
import { Menu, Transition } from "@headlessui/react";
// plane imports
import type { EditorRefApi } from "@plane/editor";
import { CheckIcon, CloseIcon } from "@plane/propel/icons";
import type { TIssuePublicComment } from "@plane/types";
import { getFileURL } from "@plane/utils/file";
// components
import { LiteTextEditor } from "@/components/editor/lite-text-editor";
import { CommentReactions } from "@/components/issues/peek-overview/comment/comment-reactions";
// helpers
import { timeAgo } from "@/helpers/date-time.helper";
// hooks
import { commentMutationOptions } from "@/queries/issue-actions";
import { EFileAssetType } from "@plane/types";
import useIsInIframe from "@/hooks/use-is-in-iframe";

const fileService = new SitesFileService();

type Props = {
  anchor: string;
  issueId: string;
  comment: TIssuePublicComment;
};

export const CommentCard = function CommentCard(props: Props) {
  const { anchor, issueId, comment } = props;
  const mutation = useMutation(commentMutationOptions(useQueryClient(), anchor, issueId));
  const { data: currentUser } = useQuery(currentUserOptions);
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));
  const { workspace: workspaceID } = publishSettings ?? {};
  const isInIframe = useIsInIframe();

  // states
  const [isEditing, setIsEditing] = useState(false);
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const showEditorRef = useRef<EditorRefApi>(null);
  const form = useForm({
    defaultValues: { comment_html: comment.comment_html },
    onSubmit: async ({ value, formApi }) => {
      if (!anchor || !issueId) return;
      await mutation.mutateAsync({ kind: "update", id: comment.id, data: value });
      formApi.reset(value);
      setIsEditing(false);
      showEditorRef.current?.setEditorValue(value.comment_html);
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const submit = (event: React.SyntheticEvent) => {
    event.preventDefault();
    void form.handleSubmit().catch(() => {
      // The mutation already reported the failed comment update.
    });
  };

  const handleDelete = () => {
    if (!anchor || !issueId) return;
    mutation.mutate({ kind: "delete", id: comment.id });
  };

  return (
    <div className="relative flex items-start space-x-3">
      <div className="relative px-1">
        {comment.actor_detail.avatar_url ? (
          <img
            src={getFileURL(comment.actor_detail.avatar_url)}
            alt={
              comment.actor_detail.is_bot ? comment.actor_detail.first_name + " Bot" : comment.actor_detail.display_name
            }
            height={30}
            width={30}
            className="grid size-7 place-items-center rounded-full border-2 border-strong-1"
          />
        ) : (
          <div
            className={`bg-gray-500 grid size-7 place-items-center rounded-full border-2 border-strong-1 text-on-color`}
          >
            {comment.actor_detail.is_bot
              ? comment?.actor_detail?.first_name?.charAt(0)
              : comment?.actor_detail?.display_name?.charAt(0)}
          </div>
        )}

        <span className="absolute -right-1 -bottom-0.5 rounded-tl-sm bg-layer-1 px-0.5 py-px">
          <MessageSquare className="size-3 text-secondary" aria-hidden="true" strokeWidth={2} />
        </span>
      </div>
      <div className="min-w-0 flex-1">
        <div>
          <div className="text-11">
            {comment.actor_detail.is_bot ? comment.actor_detail.first_name + " Bot" : comment.actor_detail.display_name}
          </div>
          <p className="mt-0.5 text-11 text-secondary">
            <>commented {timeAgo(comment.created_at)}</>
          </p>
        </div>
        <div className="issue-comments-section p-0">
          <form onSubmit={submit} className={`flex-col gap-2 ${isEditing ? "flex" : "hidden"}`}>
            <div>
              <form.Field name="comment_html">
                {(field) => (
                  <LiteTextEditor
                    editable
                    anchor={anchor}
                    workspaceId={workspaceID?.toString() ?? ""}
                    onEnterKeyPress={submit}
                    ref={editorRef}
                    id={comment.id}
                    initialValue={field.state.value}
                    value={null}
                    onChange={(comment_json, comment_html) => field.handleChange(comment_html)}
                    isSubmitting={isSubmitting}
                    showSubmitButton={false}
                    uploadFile={async (blockId, file) => {
                      const { asset_id } = await fileService.uploadAsset(
                        anchor,
                        { entity_identifier: comment.id, entity_type: EFileAssetType.COMMENT_DESCRIPTION },
                        file
                      );
                      return asset_id;
                    }}
                    displayConfig={{
                      fontSize: "small-font",
                    }}
                  />
                )}
              </form.Field>
            </div>
            <div className="flex gap-1 self-end">
              <button
                type="submit"
                disabled={isSubmitting}
                className="group shadow-md rounded-sm border border-success-strong bg-success-primary p-2 duration-300 hover:bg-success-primary"
              >
                <CheckIcon className="h-3 w-3 text-on-color" strokeWidth={2} />
              </button>
              <button
                type="button"
                disabled={isSubmitting}
                className="group shadow-md rounded-sm border border-danger-strong bg-danger-primary p-2 duration-300 hover:bg-danger-primary-hover"
                onClick={() => setIsEditing(false)}
              >
                <CloseIcon className="h-3 w-3 text-on-color" strokeWidth={2} />
              </button>
            </div>
          </form>
          <div className={`${isEditing ? "hidden" : ""}`}>
            <LiteTextEditor
              editable={false}
              anchor={anchor}
              workspaceId={workspaceID?.toString() ?? ""}
              ref={showEditorRef}
              id={comment.id}
              initialValue={comment.comment_html}
              displayConfig={{
                fontSize: "small-font",
              }}
            />
            <CommentReactions anchor={anchor} issueId={issueId} comment={comment} />
          </div>
        </div>
      </div>
      {!isInIframe && currentUser?.id === comment?.actor_detail?.id && (
        <Menu as="div" className="relative w-min text-left">
          <Menu.Button
            type="button"
            onClick={() => {}}
            className="relative grid cursor-pointer place-items-center rounded-sm p-1 text-tertiary outline-none hover:bg-layer-transparent-hover"
          >
            <MoreVertical className="size-4" strokeWidth={2} />
          </Menu.Button>

          <Transition
            as={React.Fragment}
            enter="transition ease-out duration-100"
            enterFrom="transform opacity-0 scale-95"
            enterTo="transform opacity-100 scale-100"
            leave="transition ease-in duration-75"
            leaveFrom="transform opacity-100 scale-100"
            leaveTo="transform opacity-0 scale-95"
          >
            <Menu.Items className="shadow-lg absolute right-0 z-10 mt-1 max-h-36 min-w-[8rem] origin-top-right overflow-auto overflow-y-scroll rounded-md border border-strong bg-surface-1 p-1 text-11 whitespace-nowrap focus:outline-none">
              <Menu.Item>
                {({ active }) => (
                  <div className="py-1">
                    <button
                      type="button"
                      onClick={() => {
                        form.reset({ comment_html: comment.comment_html });
                        editorRef.current?.setEditorValue(comment.comment_html);
                        setIsEditing(true);
                      }}
                      className={`w-full truncate rounded-sm px-1 py-1.5 text-left text-secondary select-none hover:bg-layer-transparent-hover ${
                        active ? "bg-layer-transparent-hover" : ""
                      }`}
                    >
                      Edit
                    </button>
                  </div>
                )}
              </Menu.Item>
              <Menu.Item>
                {({ active }) => (
                  <div className="py-1">
                    <button
                      type="button"
                      disabled={mutation.isPending}
                      onClick={handleDelete}
                      className={`w-full truncate rounded-sm px-1 py-1.5 text-left text-secondary select-none hover:bg-layer-transparent-hover ${
                        active ? "bg-layer-transparent-hover" : ""
                      }`}
                    >
                      Delete
                    </button>
                  </div>
                )}
              </Menu.Item>
            </Menu.Items>
          </Transition>
        </Menu>
      )}
    </div>
  );
};
