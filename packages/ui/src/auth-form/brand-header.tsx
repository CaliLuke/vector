/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { cloneElement } from "react";
import type { ReactElement, ReactNode } from "react";
import { PlaneLockup } from "@plane/propel/icons";

/** Keep branding shared while each application supplies its own router link. */
export function BrandHeader({ homeLink }: { homeLink: ReactElement<{ children?: ReactNode }> }) {
  return (
    <div className="sticky top-0 flex w-full flex-shrink-0 items-center justify-between gap-6">
      {cloneElement(homeLink, undefined, <PlaneLockup height={20} width={95} className="text-primary" />)}
    </div>
  );
}
