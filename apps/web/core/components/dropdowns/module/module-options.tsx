/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import type { Placement } from "@popperjs/core";

// plane imports
import { useTranslation } from "@plane/i18n";
import { ModuleIcon } from "@plane/propel/icons";
import type { IModule } from "@plane/types";
import { sortBySelectedFirst } from "@plane/utils/array";
import { CatalogueOptionsPopup } from "@/components/cycles-modules/catalogue-options-popup";
import type { CatalogueOption } from "@/components/cycles-modules/catalogue-options-popup";

interface Props {
  modules: readonly IModule[] | undefined;
  isError: boolean;
  retry: () => Promise<unknown>;
  multiple: boolean;
  placement: Placement | undefined;
  referenceElement: HTMLButtonElement | null;
  restoreButtonFocusOnClose: () => void;
  value?: string[] | string | null;
}

export const ModuleOptions = function ModuleOptions(props: Props) {
  const { modules, isError, retry, multiple, placement, referenceElement, restoreButtonFocusOnClose, value } = props;
  // refs
  // states
  const [query, setQuery] = useState("");
  // plane hooks
  const { t } = useTranslation();
  const options: CatalogueOption[] | undefined = modules?.map((moduleDetails) => {
    const moduleId = moduleDetails.id;
    return {
      value: moduleId,
      query: moduleDetails.name,
      content: (
        <div className="flex items-center gap-2">
          <ModuleIcon className="h-3 w-3 flex-shrink-0" />
          <span className="flex-grow truncate">{moduleDetails?.name}</span>
        </div>
      ),
    };
  });
  if (!multiple)
    options?.unshift({
      value: null,
      query: t("module.no_module"),
      content: (
        <div className="flex items-center gap-2">
          <ModuleIcon className="h-3 w-3 flex-shrink-0" />
          <span className="flex-grow truncate">{t("module.no_module")}</span>
        </div>
      ),
    });

  const filteredOptions = sortBySelectedFirst(
    query === "" ? options : options?.filter((o) => o.query.toLowerCase().includes(query.toLowerCase())),
    value
  );

  return (
    <CatalogueOptionsPopup
      isError={isError}
      label="modules"
      noMatchesLabel={t("common.search.no_matching_results")}
      onQueryChange={setQuery}
      options={filteredOptions}
      placement={placement}
      query={query}
      referenceElement={referenceElement}
      restoreButtonFocusOnClose={restoreButtonFocusOnClose}
      retry={retry}
    />
  );
};
