/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { Network } from "lucide-react";
// plane imports
import { Tooltip } from "@plane/propel/tooltip";
import { renderFormattedTime, renderFormattedDate, calculateTimeAgo } from "@plane/utils/datetime";
import type { TIssueActivity } from "@plane/types";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { IssueUser } from "../";
import { IssueCreatorDisplay } from "./issue-creator";

type TIssueActivityBlockComponent = {
  icon?: ReactNode;
  activity: TIssueActivity;
  ends: "top" | "bottom" | undefined;
  children: ReactNode;
  customUserName?: string;
  wrap?: boolean;
};

export function IssueActivityBlockComponent(props: TIssueActivityBlockComponent) {
  const { icon, activity, ends, children, customUserName, wrap = false } = props;
  // hooks

  const { isMobile } = usePlatformOS();
  if (!activity) return <></>;
  return (
    <div
      className={`relative flex items-center gap-3 text-caption-sm-regular ${
        ends === "top" ? `pb-2` : ends === "bottom" ? `pt-2` : `py-2`
      }`}
    >
      <div className="absolute top-0 bottom-0 left-[13px] w-px bg-layer-3" aria-hidden />
      <div className="z-[4] flex h-7 w-7 flex-shrink-0 items-center justify-center overflow-hidden rounded-lg border border-subtle bg-layer-2 text-secondary shadow-raised-100">
        {icon ? icon : <Network className="h-3.5 w-3.5" />}
      </div>
      <div className={`w-full text-secondary ${wrap ? "min-w-0 break-words" : "truncate"}`}>
        {!activity?.field && activity?.verb === "created" ? (
          <IssueCreatorDisplay activity={activity} customUserName={customUserName} />
        ) : (
          <IssueUser activity={activity} customUserName={customUserName} />
        )}
        <span> {children} </span>
        <span>
          <Tooltip
            isMobile={isMobile}
            tooltipContent={`${renderFormattedDate(activity.created_at)}, ${renderFormattedTime(activity.created_at)}`}
          >
            <span className="whitespace-nowrap text-tertiary"> {calculateTimeAgo(activity.created_at)}</span>
          </Tooltip>
        </span>
      </div>
    </div>
  );
}
