/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useParams } from "react-router";
import { EIconSize } from "@plane/constants";
import { StateGroupIcon } from "@plane/propel/icons";
import { statesOptions } from "@/queries/lookups";
import { FilterSection } from "./helpers/filter-section";
import type { FilterSelectionProps } from "./helpers/filter-section";
export function FilterState(props: FilterSelectionProps) {
  const query = useRecordCollection(statesOptions(useParams().anchor));
  const options = (query.data ?? []).map((state) => ({
    id: state.id,
    name: state.name,
    icon: <StateGroupIcon stateGroup={state.group} color={state.color} size={EIconSize.MD} />,
  }));
  return (
    <FilterSection
      {...props}
      title="State"
      options={options}
      loading={query.isPending}
      error={query.isError}
      retry={() => void query.refetch()}
    />
  );
}
