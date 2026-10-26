/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { ComponentProps } from "react";
import { DescriptionInput } from "@/components/editor/rich-text/description-input";
import type { TIssueOperations } from "./issue-detail";
import { useItemTextEditor, useItemTextSession } from "./use-item-edit-coordination";

type Props = Omit<ComponentProps<typeof DescriptionInput>, "onSubmit" | "mutationScope" | "projectId"> & {
  projectId: string;
  issueOperations: Pick<TIssueOperations, "updateText">;
};

export function IssueDescriptionInput(props: Props) {
  const session = useItemTextSession();
  return (
    <IssueDescriptionEditor
      key={`${session.key}/${props.workspaceSlug}/${props.projectId}/${props.entityId}`}
      {...props}
    />
  );
}
function IssueDescriptionEditor({ issueOperations, ...props }: Props) {
  const { workspaceSlug, projectId, entityId } = props;
  const editing = useItemTextEditor(`${workspaceSlug}/${projectId}/${entityId}`, props.setIsSubmitting);
  return (
    <DescriptionInput
      {...props}
      disabled={props.disabled || editing.locked}
      canEdit={editing.canEdit}
      canSave={editing.current}
      setIsSubmitting={editing.report}
      mutationScope={`issue-text/${workspaceSlug}/${projectId}/${entityId}`}
      onSubmit={async (value, isMigrationUpdate) => {
        await issueOperations.updateText(workspaceSlug, projectId, entityId, {
          description_html: value.description_html,
          ...(isMigrationUpdate ? { skip_activity: "true" } : {}),
        });
      }}
    />
  );
}
