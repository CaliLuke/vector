/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { cn } from "@plane/utils/common";
// hooks
import { useWorkspaceSearch } from "@/hooks/use-workspace-search";
import { usePowerK } from "@/hooks/use-power-k";
import useDebounce from "@/hooks/use-debounce";
// local imports
import type { TPowerKContext, TPowerKPageType } from "../../core/types";
import { PowerKModalNoSearchResultsCommand } from "./no-results-command";
import { PowerKModalSearchResults } from "./search-results";

type Props = {
  activePage: TPowerKPageType | null;
  context: TPowerKContext;
  isWorkspaceLevel: boolean;
  searchTerm: string;
  updateSearchTerm: (value: string) => void;
  handleSearchMenuClose?: () => void;
};

export function PowerKModalSearchMenu(props: Props) {
  const { activePage, context, isWorkspaceLevel, searchTerm, updateSearchTerm, handleSearchMenuClose } = props;
  const debouncedSearchTerm = useDebounce(searchTerm, 500);
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { togglePowerKModal } = usePowerK();
  const { status, results, retry } = useWorkspaceSearch(
    workspaceSlug,
    projectId,
    isWorkspaceLevel,
    searchTerm,
    debouncedSearchTerm,
    !activePage
  );

  if (activePage) return null;

  const handleClosePalette = () => {
    handleSearchMenuClose?.();
    togglePowerKModal(false);
  };

  return (
    <>
      {searchTerm.trim() !== "" && (
        <div className="mt-4 flex items-center justify-between gap-2 px-4">
          <h5
            className={cn("text-11 text-primary", {
              "animate-pulse": status === "loading",
            })}
          >
            Search results for{" "}
            <span className="font-medium">
              {'"'}
              {searchTerm}
              {'"'}
            </span>{" "}
            in {isWorkspaceLevel ? "workspace" : "project"}:
          </h5>
        </div>
      )}

      {searchTerm.trim() !== "" && (status === "error" || status === "denied") && (
        <div className="px-4 py-3 text-13 text-secondary" role="alert">
          {status === "denied"
            ? "Workspace search is unavailable with your current access."
            : "Search could not be loaded."}{" "}
          <button type="button" className="text-accent-primary" onClick={retry}>
            Try again
          </button>
        </div>
      )}
      {/* Show empty state only when not loading and no results */}
      {status === "empty" && (
        <PowerKModalNoSearchResultsCommand
          context={context}
          searchTerm={searchTerm}
          updateSearchTerm={updateSearchTerm}
        />
      )}

      {searchTerm.trim() !== "" && <PowerKModalSearchResults closePalette={handleClosePalette} results={results} />}
    </>
  );
}
