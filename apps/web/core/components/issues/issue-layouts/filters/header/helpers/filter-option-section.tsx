/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, type ReactNode } from "react";

import { Loader } from "@plane/ui";

import { FilterHeader } from "./filter-header";

type FilterOptionSectionState = { kind: "loading"; content?: ReactNode } | { kind: "error"; content: ReactNode };

type Props = {
  title: string;
  optionCount: number;
  renderOptions: (limit: number) => ReactNode;
  emptyMessage?: ReactNode;
  state?: FilterOptionSectionState;
};

export function FilterOptionSection({ title, optionCount, renderOptions, emptyMessage, state }: Props) {
  const [itemsToRender, setItemsToRender] = useState(5);
  const [previewEnabled, setPreviewEnabled] = useState(true);

  const handleViewToggle = () => {
    if (itemsToRender === optionCount) setItemsToRender(5);
    else setItemsToRender(optionCount);
  };

  return (
    <>
      <FilterHeader
        title={title}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {state ? (
            (state.content ?? (
              <Loader className="space-y-2">
                <Loader.Item height="20px" />
                <Loader.Item height="20px" />
                <Loader.Item height="20px" />
              </Loader>
            ))
          ) : optionCount > 0 ? (
            <>
              {renderOptions(itemsToRender)}
              {optionCount > 5 && (
                <button
                  type="button"
                  className="ml-8 text-11 font-medium text-accent-primary"
                  onClick={handleViewToggle}
                >
                  {itemsToRender === optionCount ? "View less" : "View all"}
                </button>
              )}
            </>
          ) : (
            (emptyMessage ?? <p className="text-11 text-placeholder italic">No matches found</p>)
          )}
        </div>
      )}
    </>
  );
}
