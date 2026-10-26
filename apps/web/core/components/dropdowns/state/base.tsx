/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { StateGroupIcon } from "@plane/propel/icons";
import type { IState } from "@plane/types";
import { StateDropdownControl } from "./control";
import type { StateDropdownControlProps } from "./control";

export function WorkItemStateDropdownBase(props: StateDropdownControlProps<IState>) {
  return (
    <StateDropdownControl
      {...props}
      renderIcon={(state, className) => (
        <StateGroupIcon
          stateGroup={state?.group ?? "backlog"}
          color={state?.color ?? "var(--text-color-tertiary)"}
          className={className}
          percentage={state?.order}
        />
      )}
    />
  );
}
