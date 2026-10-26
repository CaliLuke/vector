/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
// ui
// helpers
import { discussionOwnerKey } from "@/queries/issue-discussion";
import { useIssueReactions } from "@/hooks/use-issue-discussion";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TIssueServiceType } from "@plane/types";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { ReactionPicker } from "./picker";
import { buildReactionList } from "./data";
// types

export type TIssueReaction = {
  workspaceSlug: string;
  projectId: string;
  serviceType?: TIssueServiceType;
  issueId: string;
  disabled?: boolean;
  className?: string;
};

export const IssueReaction = function IssueReaction(props: TIssueReaction) {
  const { workspaceSlug, projectId, issueId, disabled = false, className = "" } = props;
  const native = useIssueReactions({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: props.serviceType,
  });
  const current = useCommittedScope(discussionOwnerKey(native.scope));
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const rows = native.query.isSuccess ? native.query.data : [];
  const react = async (reaction: string) => {
    try {
      const result = await native.mutation.mutateAsync({
        kind: "toggle",
        reaction,
      });
      if (current() && result.captured.isCurrent())
        setToast({ title: "Success!", type: TOAST_TYPE.SUCCESS, message: "Reaction saved successfully" });
    } catch {
      if (current() && native.isCurrent())
        setToast({ title: "Error!", type: TOAST_TYPE.ERROR, message: "Could not save reaction" });
    }
  };

  const reactions = buildReactionList(
    rows,
    native.scope.accountId,
    (actorId) => membersById?.get(actorId)?.member.display_name
  );

  if (native.isError)
    return (
      <div role="alert">
        Could not load reactions.{" "}
        <button type="button" onClick={() => void native.retry().catch(() => {})}>
          Retry
        </button>
      </div>
    );
  return (
    <ReactionPicker
      reactions={reactions}
      onReact={(reaction) => void react(reaction)}
      disabled={disabled || native.isPending}
      className={className}
    />
  );
};
