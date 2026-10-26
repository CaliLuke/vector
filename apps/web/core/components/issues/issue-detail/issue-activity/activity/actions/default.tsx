/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { WorkItemsIcon } from "@plane/propel/icons";
import { EInboxIssueSource } from "@plane/types";
// hooks
import { capitalizeFirstLetter } from "@plane/utils/string";
import type { TIssueActivity } from "@plane/types";
// local imports
import { IssueActivityBlockComponent } from "./";

type TIssueDefaultActivity = { activity: TIssueActivity; ends: "top" | "bottom" | undefined };

export const IssueDefaultActivity = function IssueDefaultActivity(props: TIssueDefaultActivity) {
  const { activity, ends } = props;
  // hooks

  if (!activity) return <></>;
  const source = activity.source_data?.source;

  return (
    <IssueActivityBlockComponent
      activity={activity}
      icon={<WorkItemsIcon width={14} height={14} className="text-secondary" aria-hidden="true" />}
      ends={ends}
    >
      <>
        {activity.verb === "created" ? (
          source && source !== EInboxIssueSource.IN_APP ? (
            <span>
              created the work item via{" "}
              <span className="font-medium">{capitalizeFirstLetter(source.toLowerCase() || "")}</span>.
            </span>
          ) : (
            <span> created the work item.</span>
          )
        ) : (
          <span> deleted a work item.</span>
        )}
      </>
    </IssueActivityBlockComponent>
  );
};
