/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useParams } from "react-router";

// plane ui
import { StateGroupIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TStateGroups } from "@plane/types";
// plane utils
import { cn } from "@plane/utils/common";
//hooks
import { statesOptions } from "@/queries/lookups";

type Props = {
  shouldShowBorder?: boolean;
} & (
  | {
      stateDetails: {
        name: string;
        group: TStateGroups;
      };
    }
  | {
      stateId: string;
    }
);

export const IssueBlockState = function IssueBlockState(props: Props) {
  const { shouldShowBorder = true } = props;
  const { data: states = [] } = useRecordCollection(statesOptions(useParams().anchor));
  // derived values
  const state = "stateId" in props ? states.find((item) => item.id === props.stateId) : props.stateDetails;
  if (!state) return null;

  return (
    <Tooltip tooltipHeading="State" tooltipContent={state.name}>
      <div
        className={cn("flex h-full w-full items-center justify-between gap-1 rounded-sm px-2.5 py-1 text-11", {
          "border-[0.5px] border-strong": shouldShowBorder,
        })}
      >
        <div className="flex w-full items-center gap-1.5">
          <StateGroupIcon stateGroup={state.group} />
          <div className="text-11">{state.name}</div>
        </div>
      </div>
    </Tooltip>
  );
};
