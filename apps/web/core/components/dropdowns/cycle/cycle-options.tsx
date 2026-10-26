/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import type { Placement } from "@popperjs/core";

// components
import { useTranslation } from "@plane/i18n";
import { CycleGroupIcon, CycleIcon } from "@plane/propel/icons";
import type { ICycle } from "@plane/types";
import { CatalogueOptionsPopup } from "@/components/cycles-modules/catalogue-options-popup";
import type { CatalogueOption } from "@/components/cycles-modules/catalogue-options-popup";

type CycleOptionsProps = {
  cycles: readonly ICycle[] | undefined;
  isError: boolean;
  retry: () => Promise<unknown>;
  referenceElement: HTMLButtonElement | null;
  placement: Placement | undefined;
  canRemoveCycle: boolean;
  restoreButtonFocusOnClose: () => void;
  currentCycleId?: string;
};

export const CycleOptions = function CycleOptions(props: CycleOptionsProps) {
  const {
    cycles,
    isError,
    retry,
    referenceElement,
    placement,
    canRemoveCycle,
    currentCycleId,
    restoreButtonFocusOnClose,
  } = props;
  // i18n
  const { t } = useTranslation();
  //state hooks
  const [query, setQuery] = useState("");
  const available = cycles?.filter(
    (cycle) => cycle.id !== currentCycleId && cycle.status?.toLowerCase() !== "completed"
  );
  const options: CatalogueOption[] | undefined = available?.map((cycleDetails) => {
    const cycleId = cycleDetails.id;
    const cycleStatus =
      cycleDetails.status?.toLocaleLowerCase() === "current"
        ? "current"
        : cycleDetails.status?.toLocaleLowerCase() === "upcoming"
          ? "upcoming"
          : "draft";
    return {
      value: cycleId,
      query: cycleDetails.name,
      content: (
        <div className="flex items-center gap-2">
          <CycleGroupIcon cycleGroup={cycleStatus} className="h-3.5 w-3.5 flex-shrink-0" />
          <span className="flex-grow truncate">{cycleDetails?.name}</span>
        </div>
      ),
    };
  });

  if (canRemoveCycle) {
    options?.unshift({
      value: null,
      query: t("cycle.no_cycle"),
      content: (
        <div className="flex items-center gap-2">
          <CycleIcon className="h-3 w-3 flex-shrink-0" />
          <span className="flex-grow truncate">{t("cycle.no_cycle")}</span>
        </div>
      ),
    });
  }

  const filteredOptions =
    query === "" ? options : options?.filter((o) => o.query.toLowerCase().includes(query.toLowerCase()));

  return (
    <CatalogueOptionsPopup
      isError={isError}
      label="cycles"
      noMatchesLabel={t("common.search.no_matches_found")}
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
