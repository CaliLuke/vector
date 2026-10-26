/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { SearchIcon, CloseIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";

export function FilterSelectionSearch(props: {
  value: string;
  onChange: (value: string) => void;
  autoFocus?: boolean;
  className?: string;
  inputClassName?: string;
}) {
  return (
    <div className={cn("p-2.5 pb-0", props.className ?? "bg-surface-1")}>
      <div className="flex items-center gap-1.5 rounded-sm border-[0.5px] border-subtle bg-surface-2 px-1.5 py-1 text-11">
        <SearchIcon className="text-placeholder" width={12} height={12} strokeWidth={2} />
        <input
          type="text"
          className={cn("w-full bg-surface-2 outline-none placeholder:text-placeholder", props.inputClassName)}
          placeholder="Search"
          value={props.value}
          onChange={(event) => props.onChange(event.target.value)}
          // Preserve the existing behavior for desktop filter pickers.
          // oxlint-disable-next-line jsx_a11y/no-autofocus
          autoFocus={props.autoFocus}
        />
        {props.value !== "" && (
          <button type="button" className="grid place-items-center" onClick={() => props.onChange("")}>
            <CloseIcon className="text-tertiary" height={12} width={12} strokeWidth={2} />
          </button>
        )}
      </div>
    </div>
  );
}
