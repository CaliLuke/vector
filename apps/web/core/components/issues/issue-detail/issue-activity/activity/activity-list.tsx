/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";

// helpers
import { getValidKeysFromObject } from "@plane/utils/array";
// hooks
import type { TIssueActivity } from "@plane/types";
import { useTimeLineRelationOptions } from "@/components/relations";
import { IssuePropertyActivity } from "./actions/property";
// local components
import {
  IssueDefaultActivity,
  IssueNameActivity,
  IssueDescriptionActivity,
  IssueStateActivity,
  IssueAssigneeActivity,
  IssuePriorityActivity,
  IssueEstimateActivity,
  IssueParentActivity,
  IssueRelationActivity,
  IssueDateActivity,
  IssueCycleActivity,
  IssueModuleActivity,
  IssueLabelActivity,
  IssueLinkActivity,
  IssueAttachmentActivity,
  IssueArchivedAtActivity,
  IssueInboxActivity,
} from "./actions";

type TIssueActivityItem = {
  activity: TIssueActivity;
  ends: "top" | "bottom" | undefined;
};

// These fields share the same activity props and precede relation dispatch.
const propertyActivities = new Map<string, (props: TIssueActivityItem) => ReactNode>([
  ["state", (props) => <IssueStateActivity {...props} showIssue={false} />],
  ["name", (props) => <IssueNameActivity {...props} />],
  ["description", (props) => <IssueDescriptionActivity {...props} showIssue={false} />],
  ["assignees", (props) => <IssueAssigneeActivity {...props} showIssue={false} />],
  ["priority", (props) => <IssuePriorityActivity {...props} showIssue={false} />],
  ["estimate_points", (props) => <IssueEstimateActivity {...props} showIssue={false} />],
  ["estimate_categories", (props) => <IssueEstimateActivity {...props} showIssue={false} />],
  ["estimate_time", (props) => <IssueEstimateActivity {...props} showIssue={false} />],
  // Retain history recorded before estimate fields included their type.
  ["estimate_point", (props) => <IssueEstimateActivity {...props} showIssue={false} />],
  ["parent", (props) => <IssueParentActivity {...props} showIssue={false} />],
]);

export const IssueActivityItem = function IssueActivityItem(props: TIssueActivityItem) {
  const { activity, ends } = props;
  // hooks
  const ISSUE_RELATION_OPTIONS = useTimeLineRelationOptions();
  const activityRelations = getValidKeysFromObject(ISSUE_RELATION_OPTIONS);

  const componentDefaultProps = { activity, ends };

  const activityField = activity.field;
  const renderProperty = propertyActivities.get(activityField ?? "");
  if (renderProperty) return renderProperty(componentDefaultProps);

  switch (activityField) {
    case null: // default issue creation
      return <IssueDefaultActivity {...componentDefaultProps} />;
    case activityRelations.find((field) => field === activityField):
      return <IssueRelationActivity {...componentDefaultProps} />;
    case "start_date":
    case "target_date":
      return <IssueDateActivity {...componentDefaultProps} showIssue={false} />;
    case "cycles":
      return <IssueCycleActivity {...componentDefaultProps} />;
    case "modules":
      return <IssueModuleActivity {...componentDefaultProps} />;
    case "labels":
      return <IssueLabelActivity {...componentDefaultProps} showIssue={false} />;
    case "link":
      return <IssueLinkActivity {...componentDefaultProps} showIssue={false} />;
    case "attachment":
      return <IssueAttachmentActivity {...componentDefaultProps} showIssue={false} />;
    case "archived_at":
      return <IssueArchivedAtActivity {...componentDefaultProps} />;
    case "intake":
    case "inbox":
      return <IssueInboxActivity {...componentDefaultProps} />;
    default:
      return <IssuePropertyActivity {...componentDefaultProps} />;
  }
};
