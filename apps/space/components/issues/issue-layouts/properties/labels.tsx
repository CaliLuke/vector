/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router";

import { LabelPropertyIcon } from "@plane/propel/icons";
// plane imports
import { Tooltip } from "@plane/propel/tooltip";
// hooks
import { labelsOptions } from "@/queries/lookups";

type Props = {
  labelIds: string[];
  shouldShowLabel?: boolean;
};

export const IssueBlockLabels = function IssueBlockLabels({ labelIds, shouldShowLabel = false }: Props) {
  const { data: labels = [] } = useQuery(labelsOptions(useParams().anchor));

  const selectedLabels = labels.filter((item) => labelIds.includes(item.id));

  const labelsString = selectedLabels.length > 0 ? selectedLabels.map((label) => label.name).join(", ") : "No Labels";

  if (selectedLabels.length <= 0)
    return (
      <Tooltip position="top" tooltipHeading="Labels" tooltipContent="None">
        <div
          className={`flex h-full items-center justify-center gap-2 rounded-sm border-[0.5px] border-strong px-2.5 py-1 text-11`}
        >
          <LabelPropertyIcon className="h-3.5 w-3.5" strokeWidth={2} />
          {shouldShowLabel && <span>No Labels</span>}
        </div>
      </Tooltip>
    );

  return (
    <div className="flex h-5 w-full flex-wrap items-center gap-2 overflow-hidden">
      {selectedLabels.length <= 2 ? (
        <>
          {selectedLabels.map((label) => (
            <Tooltip key={label.id} position="top" tooltipHeading="Labels" tooltipContent={label?.name ?? ""}>
              <div
                key={label?.id}
                className={`flex h-full max-w-full flex-shrink-0 items-center overflow-hidden rounded-sm border-[0.5px] border-strong px-2.5 py-1 text-11`}
              >
                <div className="flex max-w-full items-center gap-1.5 overflow-hidden text-secondary">
                  <span
                    className="h-2 w-2 flex-shrink-0 rounded-full"
                    style={{
                      backgroundColor: label?.color ?? "#000000",
                    }}
                  />
                  <div className="line-clamp-1 inline-block w-auto max-w-[100px] truncate">{label?.name}</div>
                </div>
              </div>
            </Tooltip>
          ))}
        </>
      ) : (
        <div
          className={`cursor-not-allowed" flex h-full flex-shrink-0 items-center rounded-sm border-[0.5px] border-strong px-2.5 py-1 text-11`}
        >
          <Tooltip position="top" tooltipHeading="Labels" tooltipContent={labelsString}>
            <div className="flex h-full items-center gap-1.5 text-secondary">
              <span className="h-2 w-2 flex-shrink-0 rounded-full bg-accent-primary" />
              {`${selectedLabels.length} Labels`}
            </div>
          </Tooltip>
        </div>
      )}
    </div>
  );
};
