/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState, type ReactNode } from "react";

import { CloseIcon, SearchIcon } from "@plane/propel/icons";
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  children: (searchQuery: string) => ReactNode;
};

export function FilterSelectionPanel({ children }: Props) {
  const [searchQuery, setSearchQuery] = useState("");
  const searchInputRef = useRef<HTMLInputElement>(null);
  const { isMobile } = usePlatformOS();

  useEffect(() => {
    if (!isMobile) searchInputRef.current?.focus();
  }, [isMobile]);

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <div className="bg-surface-1 p-2.5 pb-0">
        <div className="flex items-center gap-1.5 rounded-sm border-[0.5px] border-subtle bg-surface-2 px-1.5 py-1 text-11">
          <SearchIcon className="text-placeholder" width={12} height={12} strokeWidth={2} />
          <input
            ref={searchInputRef}
            type="text"
            className="w-full bg-surface-2 outline-none placeholder:text-placeholder"
            placeholder="Search"
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
          />
          {searchQuery !== "" && (
            <button type="button" className="grid place-items-center" onClick={() => setSearchQuery("")}>
              <CloseIcon className="text-tertiary" height={12} width={12} strokeWidth={2} />
            </button>
          )}
        </div>
      </div>
      <div className="vertical-scrollbar scrollbar-sm h-full w-full divide-y divide-subtle-1 overflow-y-auto px-2.5">
        {children(searchQuery)}
      </div>
    </div>
  );
}
