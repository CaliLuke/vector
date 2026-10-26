/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { publishSettingsOptions } from "@/queries/publishing";
// plane imports
import { RichTextEditor } from "@/components/editor/rich-text-editor";
// types

import type { IPublicIssue } from "@plane/types";
// local imports
import { IssueReactions } from "./issue-reaction";

type Props = {
  anchor: string;
  issueDetails: IPublicIssue;
};

export const PeekOverviewIssueDetails = function PeekOverviewIssueDetails(props: Props) {
  const { anchor, issueDetails } = props;
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));
  const { project_details, workspace: workspaceID } = publishSettings ?? {};
  // derived values
  const description = issueDetails.description_html;

  return (
    <div className="space-y-2">
      <h6 className="text-14 font-medium text-placeholder">
        {project_details?.identifier}-{issueDetails?.sequence_id}
      </h6>
      <h4 className="text-20 font-medium break-words">{issueDetails.name}</h4>
      {description && description !== "" && description !== "<p></p>" && (
        <RichTextEditor
          editable={false}
          anchor={anchor}
          id={issueDetails.id}
          initialValue={description}
          workspaceId={workspaceID?.toString() ?? ""}
        />
      )}
      <IssueReactions issue={issueDetails} anchor={anchor} />
    </div>
  );
};
