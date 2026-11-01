/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useTranslation } from "@plane/i18n";
import { ChevronDownIcon } from "@plane/propel/icons";
import { FiltersDropdown } from "./helpers/dropdown";

export function MobileDisplayOptions({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-grow items-center justify-center border-l border-subtle text-13 text-secondary">
      <FiltersDropdown
        title={t("common.display")}
        placement="bottom-end"
        menuButton={
          <span className="flex items-center text-13 text-secondary">
            {t("common.display")}
            <ChevronDownIcon className="ml-2 h-4 w-4 text-secondary" />
          </span>
        }
      >
        {children}
      </FiltersDropdown>
    </div>
  );
}
