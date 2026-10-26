/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import type { IPublicIssue } from "@plane/types";
import { Loader } from "@plane/ui";
import { publishSettingsOptions } from "@/queries/publishing";
import type { IPeekMode } from "@/types/issue";
import { PeekOverviewHeader } from "./header";
import { PeekOverviewIssueActivity } from "./issue-activity";
import { PeekOverviewIssueDetails } from "./issue-details";
import { PeekOverviewIssueProperties } from "./issue-properties";

type Props = {
  anchor: string;
  handleClose: () => void;
  peekMode: IPeekMode;
  setPeekMode: (mode: IPeekMode) => void;
  issueDetails: IPublicIssue | undefined;
};
function ContentSkeleton() {
  return (
    <Loader className="space-y-3 px-6">
      <Loader.Item height="30px" />
      <Loader.Item height="20px" />
      <Loader.Item height="20px" />
      <Loader.Item height="20px" />
    </Loader>
  );
}
export function PeekView({ anchor, handleClose, peekMode, setPeekMode, issueDetails }: Props) {
  const { data: settings } = useQuery(publishSettingsOptions(anchor));
  const full = peekMode === "full";
  return (
    <div
      className={
        full
          ? "grid size-full grid-cols-10 divide-x divide-subtle-1 overflow-hidden"
          : "flex size-full flex-col overflow-hidden"
      }
    >
      <div
        className={
          full ? "col-span-7 flex size-full flex-col overflow-hidden" : "flex size-full flex-col overflow-hidden"
        }
      >
        <div className="w-full p-5">
          <PeekOverviewHeader
            handleClose={handleClose}
            issueDetails={issueDetails}
            peekMode={peekMode}
            setPeekMode={setPeekMode}
          />
        </div>
        {issueDetails ? (
          <div className="size-full overflow-y-auto px-6">
            <PeekOverviewIssueDetails anchor={anchor} issueDetails={issueDetails} />
            {!full && (
              <div className="mt-6 w-full">
                <PeekOverviewIssueProperties issueDetails={issueDetails} />
              </div>
            )}
            <div className="my-5 w-full border-t border-subtle" />
            {settings?.is_comments_enabled && (
              <div className="w-full pb-5">
                <PeekOverviewIssueActivity anchor={anchor} issueDetails={issueDetails} />
              </div>
            )}
          </div>
        ) : (
          <ContentSkeleton />
        )}
      </div>
      {full && (
        <div className="col-span-3 size-full overflow-y-auto px-6 py-5">
          {issueDetails ? <PeekOverviewIssueProperties issueDetails={issueDetails} /> : <ContentSkeleton />}
        </div>
      )}
    </div>
  );
}
