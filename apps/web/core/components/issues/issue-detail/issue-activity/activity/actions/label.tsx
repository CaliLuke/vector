/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { LabelPropertyIcon } from "@plane/propel/icons";
// hooks
import type { TIssueActivity } from "@plane/types";
import { useProjectLabels } from "@/hooks/use-labels";
// components
import { IssueActivityBlockComponent, IssueLink, LabelActivityChip } from "./";

type TIssueLabelActivity = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssueLabelActivity = function IssueLabelActivity(props: TIssueLabelActivity) {
  const { activity, showIssue = true, ends } = props;
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // hooks
  const { getLabelById } = useProjectLabels(workspaceSlug, projectId);

  const oldLabelColor = getLabelById(activity?.old_identifier ?? "")?.color;
  const newLabelColor = getLabelById(activity?.new_identifier ?? "")?.color;

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<LabelPropertyIcon height={14} width={14} className="text-secondary" />}
      activity={activity}
      ends={ends}
    >
      <>
        {activity.old_value === "" ? `added a new label ` : `removed the label `}
        <LabelActivityChip
          name={activity.old_value === "" ? activity.new_value : activity.old_value}
          color={activity.old_value === "" ? newLabelColor : oldLabelColor}
        />
        {showIssue && (activity.old_value === "" ? ` to ` : ` from `)}
        {showIssue && <IssueLink activity={activity} />}
      </>
    </IssueActivityBlockComponent>
  );
};
