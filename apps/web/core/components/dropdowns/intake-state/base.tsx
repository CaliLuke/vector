/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { IntakeStateGroupIcon } from "@plane/propel/icons";
import type { IIntakeState } from "@plane/types";
import { StateDropdownControl } from "../state/control";
import type { StateDropdownControlProps } from "../state/control";

export function WorkItemStateDropdownBase(props: StateDropdownControlProps<IIntakeState>) {
  return (
    <StateDropdownControl
      {...props}
      fallbackToFirstState
      renderIcon={(state, className) => (
        <IntakeStateGroupIcon
          stateGroup={state?.group ?? "triage"}
          color={state?.color ?? "var(--text-color-tertiary)"}
          className={className}
        />
      )}
    />
  );
}
