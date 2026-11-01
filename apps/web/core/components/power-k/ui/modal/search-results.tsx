/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { Command } from "cmdk";

// plane imports
import type { IWorkspaceSearchResults } from "@plane/types";
// hooks

// helpers
import { PowerKModalCommandItem } from "./command-item";
import { getPowerKSearchSection, isPowerKSearchResultKey } from "./search-results-map";

type Props = {
  closePalette: () => void;
  results: IWorkspaceSearchResults;
};

export function PowerKModalSearchResults(props: Props) {
  const { closePalette, results } = props;
  // router
  const router = useRouter();
  const { projectId } = useParams({ strict: false });
  // derived values

  return (
    <>
      {Object.keys(results.results)
        .filter(isPowerKSearchResultKey)
        .map((key) => {
          const section = getPowerKSearchSection(key, results.results, projectId);
          if (!section.items.length) return null;
          return (
            <Command.Group key={section.key} heading={section.title} forceMount>
              {section.items.map((item) => {
                return (
                  <PowerKModalCommandItem
                    key={item.id}
                    forceMount
                    label={item.label}
                    icon={section.icon}
                    onSelect={() => {
                      closePalette();
                      void router.navigate({ href: item.path });
                    }}
                    value={item.value}
                  />
                );
              })}
            </Command.Group>
          );
        })}
    </>
  );
}
