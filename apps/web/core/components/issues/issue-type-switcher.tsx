/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useEffectEvent, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { ArrowLeftRight } from "lucide-react";
import type { TIssue } from "@plane/types";
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useWorkItemTypeCatalog } from "@/hooks/use-work-item-type-catalog";
import { TypeIcon } from "@/components/work-item-types/controls";
import { IssueIdentifier } from "./issue-detail/issue-identifier";
import { CreateUpdateIssueModal } from "./issue-modal/modal";

export type TIssueTypeSwitcherProps = {
  workspaceSlug: string;
  issueId: string;
  disabled: boolean;
  onEditorOpenChange?: (open: boolean) => void;
};
export const IssueTypeSwitcher = function IssueTypeSwitcher(props: TIssueTypeSwitcherProps) {
  const accountId = useCurrentUserId();
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const record = useWorkItemDetailRecord();
  const issue = record?.issueId === props.issueId ? record.issue : undefined;
  const project = useWorkspaceProjectDetails(props.workspaceSlug).projects?.find(
    (projectRow) => projectRow.id === issue?.project_id
  );
  if (!issue?.project_id) return null;
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2">
      <IssueIdentifier
        projectId={issue.project_id}
        projectIdentifier={project?.identifier ?? ""}
        issueSequenceId={issue.sequence_id}
        issueTypeId={issue.type_id}
        size="md"
        enableClickToCopyIdentifier
      />
      <IssueTypeIdentity
        key={`${accountId}/${generation}/${props.workspaceSlug}/${issue.project_id}/${issue.id}`}
        {...props}
        item={issue}
      />
    </div>
  );
};
function IssueTypeIdentity({
  workspaceSlug,
  item,
  disabled,
  onEditorOpenChange,
}: TIssueTypeSwitcherProps & { item: TIssue }) {
  const { catalog, permissions } = useWorkItemTypeCatalog(workspaceSlug, item.project_id ?? undefined);
  if (permissions.isPending) return <span className="text-12 text-tertiary">Loading type…</span>;
  if (permissions.isError || permissions.projectRole === undefined) return null;
  if (catalog.isError)
    return (
      <button type="button" className="text-12 text-secondary" onClick={() => void catalog.refetch()}>
        Retry loading type
      </button>
    );
  if (!catalog.isSuccess) return <span className="text-12 text-tertiary">Loading type…</span>;
  if (!catalog.data.enabled) return null;
  const type = catalog.data.types.find((entry) => entry.id === item.type_id);
  if (!type) return <span className="text-12 text-secondary">Unavailable type</span>;
  return (
    <TypeEntry
      type={type}
      item={item}
      editable={!disabled && permissions.can([15, 20])}
      onEditorOpenChange={onEditorOpenChange}
    />
  );
}
function TypeEntry({
  type,
  item,
  editable,
  onEditorOpenChange,
}: {
  type: { name: string; icon: string; color: string };
  item: TIssue;
  editable: boolean;
  onEditorOpenChange?: (open: boolean) => void;
}) {
  const [open, setOpen] = useState(false);
  const visible = open && editable;
  const notify = useEffectEvent((value: boolean) => onEditorOpenChange?.(value));
  useEffect(() => {
    notify(visible);
    return () => notify(false);
  }, [visible]);
  const identity = (
    <>
      <TypeIcon type={type} />
      <span className="max-w-48 truncate">{type.name}</span>
    </>
  );
  return (
    <>
      {editable ? (
        <button
          type="button"
          aria-label={`Change work item type: ${type.name}`}
          onClick={() => setOpen(true)}
          className="group flex min-w-0 items-center gap-1.5 rounded px-1.5 py-1 text-13 hover:bg-layer-2 focus-visible:outline-2 focus-visible:outline-accent-strong"
        >
          {identity}
          <ArrowLeftRight
            aria-hidden
            className="size-3.5 opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100"
          />
        </button>
      ) : (
        <span className="flex min-w-0 items-center gap-1.5 text-13">{identity}</span>
      )}
      {visible && (
        <CreateUpdateIssueModal
          isOpen
          onClose={() => setOpen(false)}
          data={{ id: item.id, project_id: item.project_id }}
          withDraftIssueWrapper={false}
          isProjectSelectionDisabled
        />
      )}
    </>
  );
}
