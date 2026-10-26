/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { publishSettingsOptions } from "@/queries/publishing";
import { Link, useParams, useSearchParams } from "react-router";

// plane types
import { Tooltip } from "@plane/propel/tooltip";
import type { IIssueDisplayProperties } from "@plane/types";
// plane ui
// plane utils
import { cn } from "@plane/utils/common";
// components
import { WithDisplayPropertiesHOC } from "@/components/issues/issue-layouts/with-display-properties-HOC";
// helpers

// hooks

//

import type { IPublicIssue } from "@plane/types";
import { IssueProperties } from "../properties/all-properties";

import { BlockReactions } from "./block-reactions";

interface IssueBlockProps {
  issue: IPublicIssue;
  groupId: string;
  displayProperties: IIssueDisplayProperties | undefined;
}

interface IssueDetailsBlockProps {
  issue: IPublicIssue;
  displayProperties: IIssueDisplayProperties | undefined;
}

const KanbanIssueDetailsBlock = function KanbanIssueDetailsBlock(props: IssueDetailsBlockProps) {
  const { issue, displayProperties } = props;
  const { anchor } = useParams();
  // hooks
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));
  const { project_details } = publishSettings ?? {};

  return (
    <div className="space-y-2 px-3 py-2">
      <WithDisplayPropertiesHOC displayProperties={displayProperties || {}} displayPropertyKey="key">
        <div className="relative">
          <div className="line-clamp-1 text-11 text-tertiary">
            {project_details?.identifier}-{issue.sequence_id}
          </div>
        </div>
      </WithDisplayPropertiesHOC>

      <div className="mb-1.5 line-clamp-1 w-full text-13 text-primary">
        <Tooltip tooltipContent={issue.name}>
          <span>{issue.name}</span>
        </Tooltip>
      </div>

      <IssueProperties
        className="flex flex-wrap items-center gap-2 pt-1.5 whitespace-nowrap text-tertiary"
        issue={issue}
        displayProperties={displayProperties}
      />
    </div>
  );
};

export const KanbanIssueBlock = function KanbanIssueBlock(props: IssueBlockProps) {
  const { issue, groupId, displayProperties } = props;
  const issueId = issue.id;
  const [searchParams] = useSearchParams();
  // query params
  // hooks
  const query = new URLSearchParams(searchParams);
  query.set("peekId", issue.id);
  const queryParam = query.toString();
  const isPeeked = searchParams.get("peekId") === issue.id;

  return (
    <div className={cn("group/kanban-block relative p-1.5")}>
      <div
        className={cn(
          "relative block w-full rounded-lg border border-subtle bg-layer-2 text-13 transition-all hover:bg-layer-2-hover",
          {
            "border-accent-strong hover:border-accent-strong": isPeeked,
          }
        )}
      >
        <Link id={`issue_${issueId}_${groupId}`} className="w-full" to={`?${queryParam}`}>
          <KanbanIssueDetailsBlock issue={issue} displayProperties={displayProperties} />
        </Link>
        <BlockReactions issue={issue} />
      </div>
    </div>
  );
};
