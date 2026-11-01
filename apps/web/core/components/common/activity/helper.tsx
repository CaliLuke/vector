/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC, ReactNode } from "react";
import {
  RotateCcw,
  Network,
  Inbox,
  AlignLeft,
  Paperclip,
  Type,
  FileText,
  Hash,
  Clock,
  Bell,
  GitBranch,
  Timer,
  ListTodo,
  Layers,
} from "lucide-react";
// components

import {
  LinkIcon,
  ArchiveIcon,
  CycleIcon,
  GlobeIcon,
  DueDatePropertyIcon,
  EstimatePropertyIcon,
  GridLayoutIcon,
  IntakeIcon,
  LabelPropertyIcon,
  MembersPropertyIcon,
  ModuleIcon,
  PriorityPropertyIcon,
  StartDatePropertyIcon,
  StatePropertyIcon,
} from "@plane/propel/icons";
import type { IWorkspace, TProjectActivity } from "@plane/types";

type ActivityIconMap = {
  [key: string]: FC<{ className?: string }>;
};
export const iconsMap: ActivityIconMap = {
  priority: PriorityPropertyIcon,
  archived_at: ArchiveIcon,
  restored: RotateCcw,
  link: LinkIcon,
  start_date: StartDatePropertyIcon,
  target_date: DueDatePropertyIcon,
  label: LabelPropertyIcon,
  inbox: Inbox,
  description: AlignLeft,
  assignee: MembersPropertyIcon,
  attachment: Paperclip,
  name: Type,
  state: StatePropertyIcon,
  estimate: EstimatePropertyIcon,
  cycle: CycleIcon,
  module: ModuleIcon,
  page: FileText,
  network: GlobeIcon,
  identifier: Hash,
  timezone: Clock,
  is_project_updates_enabled: Bell,
  is_epic_enabled: GridLayoutIcon,
  is_workflow_enabled: GitBranch,
  is_time_tracking_enabled: Timer,
  is_issue_type_enabled: ListTodo,
  default: Network,
  module_view: ModuleIcon,
  cycle_view: CycleIcon,
  issue_views_view: Layers,
  page_view: FileText,
  intake_view: IntakeIcon,
};

type ActivityMessage = { message: ReactNode; customUserName?: string };
type ActivityMessageRenderer = (activity: TProjectActivity, workspaceDetail: IWorkspace | undefined) => ActivityMessage;
function getBooleanActionText(value: string | undefined, verb: string) {
  if (value === "true") return "enabled";
  if (value === "false") return "disabled";
  return verb;
}
const renderProjectViewMessage: ActivityMessageRenderer = (activity) => {
  const { field: activityType, new_value: newValue, verb } = activity;
  return {
    message: (
      <>
        {getBooleanActionText(newValue, verb)} {(activityType ?? "").replace(/_view$/, "").replace(/_/g, " ")} view
      </>
    ),
  };
};

const messageRenderers = new Map<string, ActivityMessageRenderer>([
  [
    "priority",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: (
          <>
            set the priority to <span className="font-medium text-primary">{newValue || "none"}</span>
          </>
        ),
      };
    },
  ],
  [
    "archived_at",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: newValue === "restore" ? "restored the project" : "archived the project",
        customUserName: newValue === "archive" ? "Vector" : undefined,
      };
    },
  ],
  [
    "name",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: (
          <>
            renamed the project to <span className="font-medium text-primary">{newValue}</span>
          </>
        ),
      };
    },
  ],
  [
    "description",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: newValue ? "updated the project description" : "removed the project description",
      };
    },
  ],
  [
    "start_date",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: (
          <>
            {newValue ? (
              <>
                set the start date to <span className="font-medium text-primary">{newValue}</span>
              </>
            ) : (
              "removed the start date"
            )}
          </>
        ),
      };
    },
  ],
  [
    "target_date",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: (
          <>
            {newValue ? (
              <>
                set the target date to <span className="font-medium text-primary">{newValue}</span>
              </>
            ) : (
              "removed the target date"
            )}
          </>
        ),
      };
    },
  ],
  [
    "state",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: (
          <>
            set the state to <span className="font-medium text-primary">{newValue || "none"}</span>
          </>
        ),
      };
    },
  ],
  [
    "estimate",
    (activity) => {
      const { new_value: newValue, old_value: oldValue } = activity;
      return {
        message: (
          <>
            {newValue ? (
              <>
                set the estimate point to <span className="font-medium text-primary">{newValue}</span>
              </>
            ) : (
              <>
                removed the estimate point
                {oldValue && (
                  <>
                    {" "}
                    <span className="font-medium text-primary">{oldValue}</span>
                  </>
                )}
              </>
            )}
          </>
        ),
      };
    },
  ],
  [
    "cycles",
    (activity, workspaceDetail) => {
      const { verb } = activity;
      return {
        message: (
          <>
            <span>
              {verb} this project {verb === "removed" ? "from" : "to"} the cycle{" "}
            </span>
            {verb !== "removed" ? (
              <a
                href={`/${workspaceDetail?.slug}/projects/${activity.project}/cycles/${activity.new_identifier}`}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex font-medium text-primary"
              >
                {activity.new_value}
              </a>
            ) : (
              <span className="font-medium text-primary">{activity.old_value || "Unknown cycle"}</span>
            )}
          </>
        ),
      };
    },
  ],
  [
    "modules",
    (activity) => {
      const { new_value: newValue, old_value: oldValue, verb } = activity;
      return {
        message: (
          <>
            <span>
              {verb} this project {verb === "removed" ? "from" : "to"} the module{" "}
            </span>
            <span className="font-medium text-primary">
              {verb === "removed" ? oldValue : newValue || "Unknown module"}
            </span>
          </>
        ),
      };
    },
  ],
  [
    "labels",
    (activity) => {
      const { new_value: newValue, old_value: oldValue, verb } = activity;
      return {
        message: (
          <>
            {verb} the label{" "}
            <span className="font-medium text-primary">{newValue || oldValue || "Untitled label"}</span>
          </>
        ),
      };
    },
  ],
  [
    "inbox",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: <>{newValue ? "enabled" : "disabled"} inbox</>,
      };
    },
  ],
  [
    "page",
    (activity) => {
      const { new_value: newValue, old_value: oldValue } = activity;
      return {
        message: (
          <>
            {newValue ? "created" : "removed"} the project page{" "}
            <span className="font-medium text-primary">{newValue || oldValue || "Untitled page"}</span>
          </>
        ),
      };
    },
  ],
  [
    "network",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: <>{newValue ? "enabled" : "disabled"} network access</>,
      };
    },
  ],
  [
    "identifier",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: (
          <>
            updated project identifier to <span className="font-medium text-primary">{newValue || "none"}</span>
          </>
        ),
      };
    },
  ],
  [
    "timezone",
    (activity) => {
      const { new_value: newValue } = activity;
      return {
        message: (
          <>
            changed project timezone to <span className="font-medium text-primary">{newValue || "default"}</span>
          </>
        ),
      };
    },
  ],
  ...["module_view", "cycle_view", "issue_views_view", "page_view", "intake_view"].map(
    (key): [string, ActivityMessageRenderer] => [key, renderProjectViewMessage]
  ),
  [
    "is_project_updates_enabled",
    (activity) => {
      const { new_value: newValue, verb } = activity;
      return {
        message: <>{getBooleanActionText(newValue, verb)} project updates</>,
      };
    },
  ],
  [
    "is_epic_enabled",
    (activity) => {
      const { new_value: newValue, verb } = activity;
      return {
        message: <>{getBooleanActionText(newValue, verb)} epics</>,
      };
    },
  ],
  [
    "is_workflow_enabled",
    (activity) => {
      const { new_value: newValue, verb } = activity;
      return {
        message: <>{getBooleanActionText(newValue, verb)} custom workflow</>,
      };
    },
  ],
  [
    "is_time_tracking_enabled",
    (activity) => {
      const { new_value: newValue, verb } = activity;
      return {
        message: <>{getBooleanActionText(newValue, verb)} time tracking</>,
      };
    },
  ],
  [
    "is_issue_type_enabled",
    (activity) => {
      const { new_value: newValue, verb } = activity;
      return {
        message: <>{getBooleanActionText(newValue, verb)} work item types</>,
      };
    },
  ],
]);
export const messages = (activity: TProjectActivity, workspaceDetail: IWorkspace | undefined): ActivityMessage => {
  const render = messageRenderers.get(activity.field ?? "");
  return render
    ? render(activity, workspaceDetail)
    : { message: `${activity.verb} ${activity.field?.replace(/_/g, " ")} ` };
};
