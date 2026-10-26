/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { CloseIcon } from "@plane/propel/icons";

type AppliedStatusChip = {
  value: string;
  label: string;
  className: string;
  icon?: ReactNode;
};

type Props = {
  chips: readonly AppliedStatusChip[];
  editable: boolean | undefined;
  onRemove: (value: string) => void;
};

export type AppliedStatusFilterProps = {
  handleRemove: (value: string) => void;
  values: string[];
  editable: boolean | undefined;
};

export function AppliedStatusChips({ chips, editable, onRemove }: Props) {
  return chips.map((chip) => (
    <div key={chip.value} className={chip.className}>
      {chip.icon}
      {chip.label}
      {editable && (
        <button
          type="button"
          className="grid place-items-center text-tertiary hover:text-secondary"
          onClick={() => onRemove(chip.value)}
        >
          <CloseIcon height={10} width={10} strokeWidth={2} />
        </button>
      )}
    </div>
  ));
}
