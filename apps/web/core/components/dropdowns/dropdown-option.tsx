/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Combobox } from "@headlessui/react";
import { CheckIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";

type Props = { value: string | null | undefined; children: React.ReactNode };

export function DropdownOption({ value, children }: Props) {
  return (
    <Combobox.Option
      value={value}
      className={({ active, selected }) =>
        cn(
          "flex w-full cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none",
          active && "bg-layer-transparent-hover",
          selected ? "text-primary" : "text-secondary"
        )
      }
    >
      {({ selected }) => (
        <>
          <span className="flex-grow truncate">{children}</span>
          {selected && <CheckIcon className="h-3.5 w-3.5 flex-shrink-0" />}
        </>
      )}
    </Combobox.Option>
  );
}
