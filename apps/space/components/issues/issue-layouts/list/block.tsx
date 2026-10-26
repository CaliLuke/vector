/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IPublicIssue } from "@plane/types";
import { useQuery } from "@tanstack/react-query";
import { publishSettingsOptions } from "@/queries/publishing";
import { Link, useParams, useSearchParams } from "react-router";
import { useRef } from "react";

// plane types
import { Tooltip } from "@plane/propel/tooltip";
import type { IIssueDisplayProperties } from "@plane/types";
// plane ui
// plane utils
import { cn } from "@plane/utils/common";
// helpers

// hooks

//
import { IssueProperties } from "../properties/all-properties";

interface IssueBlockProps {
  issue: IPublicIssue;
  displayProperties: IIssueDisplayProperties | undefined;
}

export const IssueBlock = function IssueBlock(props: IssueBlockProps) {
  const { anchor } = useParams();
  const { issue, displayProperties } = props;
  const [searchParams] = useSearchParams();
  // query params
  // ref
  const issueRef = useRef<HTMLDivElement | null>(null);
  // hooks
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));
  const { project_details } = publishSettings ?? {};
  const query = new URLSearchParams(searchParams);
  query.set("peekId", issue.id);
  const queryParam = query.toString();
  const isPeeked = searchParams.get("peekId") === issue.id;

  const projectIdentifier = project_details?.identifier;

  return (
    <div
      ref={issueRef}
      className={cn(
        "group/list-block relative flex min-h-11 flex-col gap-3 border border-transparent border-b-subtle p-3 pl-1.5 text-13 transition-colors hover:bg-layer-transparent-hover md:flex-row md:items-center",
        {
          "border-accent-strong!": isPeeked,
          "last:border-b-transparent": !isPeeked,
        }
      )}
    >
      <div className="flex w-full truncate">
        <div className="flex grow items-center gap-0.5 truncate">
          <div className="flex items-center gap-1">
            {displayProperties && displayProperties?.key && (
              <div className="shrink-0 px-4 text-11 font-medium text-tertiary">
                {projectIdentifier}-{issue.sequence_id}
              </div>
            )}
          </div>

          <Link
            id={`issue-${issue.id}`}
            to={`?${queryParam}`}
            className="w-full cursor-pointer truncate text-13 text-primary"
          >
            <Tooltip tooltipContent={issue.name} position="top-start">
              <p className="truncate">{issue.name}</p>
            </Tooltip>
          </Link>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <IssueProperties
          className="relative flex flex-wrap items-center gap-2 whitespace-nowrap md:shrink-0 md:grow"
          issue={issue}
          displayProperties={displayProperties}
        />
      </div>
    </div>
  );
};
