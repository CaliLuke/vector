/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useParams } from "react-router";

// plane imports
import { EIconSize } from "@plane/constants";
import { CloseIcon, StateGroupIcon } from "@plane/propel/icons";
// hooks
import { statesOptions } from "@/queries/lookups";

type Props = {
  handleRemove: (val: string) => void;
  values: string[];
};

export const AppliedStateFilters = function AppliedStateFilters(props: Props) {
  const { handleRemove, values } = props;

  const { data: states = [] } = useRecordCollection(statesOptions(useParams().anchor));

  return (
    <>
      {states
        .filter((state) => values.includes(state.id))
        .map((stateDetails) => (
          <div key={stateDetails.id} className="flex items-center gap-1 rounded-sm bg-layer-3 p-1 text-11">
            <StateGroupIcon color={stateDetails.color} stateGroup={stateDetails.group} size={EIconSize.SM} />
            {stateDetails.name}
            <button
              type="button"
              className="grid place-items-center text-tertiary hover:text-secondary"
              onClick={() => handleRemove(stateDetails.id)}
            >
              <CloseIcon height={10} width={10} strokeWidth={2} />
            </button>
          </div>
        ))}
    </>
  );
};
