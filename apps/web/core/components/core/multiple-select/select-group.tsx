/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { useLocation } from "@tanstack/react-router";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
// hooks
import type { TSelectionHelper } from "@/hooks/use-multiple-select";
import { useMultipleSelect } from "@/hooks/use-multiple-select";

type Props = {
  children: (helpers: TSelectionHelper) => React.ReactNode;
  containerRef: React.MutableRefObject<HTMLElement | null>;
  disabled?: boolean;
  entities: Record<string, string[]>; // { groupID: entityIds[] }
};

export function MultipleSelectGroup(props: Props) {
  const pathname = useLocation({ select: (location) => location.pathname });
  const generation = useIdentityGeneration(useQueryClient());
  return <SelectionGroup key={JSON.stringify([generation, pathname, props.disabled ?? false])} {...props} />;
}

// Render callbacks still read observable issue/layout data until those owners migrate.
const SelectionGroup = function SelectionGroup(props: Props) {
  const { children, containerRef, disabled = false, entities } = props;

  const helpers = useMultipleSelect({
    containerRef,
    disabled,
    entities,
  });

  return <>{children(helpers)}</>;
};
