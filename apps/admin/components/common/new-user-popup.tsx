/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { instanceInfoOptions } from "@/queries/instance";
import { Link } from "react-router";
import { useTheme as useNextTheme } from "next-themes";
// ui
import { Button, getButtonStyling } from "@plane/propel/button";
import { resolveGeneralTheme } from "@plane/utils/theme-legacy";
// hooks
import TakeoffIconDark from "@/app/assets/logos/takeoff-icon-dark.svg?url";
import TakeoffIconLight from "@/app/assets/logos/takeoff-icon-light.svg?url";
// icons

export const NewUserPopup = function NewUserPopup() {
  // hooks
  const { data } = useQuery(instanceInfoOptions);
  const [dismissed, setDismissed] = useState(false);
  // theme
  const { resolvedTheme } = useNextTheme();

  if (!data || data.instance.workspaces_exist || dismissed) return <></>;
  return (
    <div className="shadow-md absolute right-8 bottom-8 w-96 rounded-lg border border-subtle bg-surface-1 p-6">
      <div className="flex gap-4">
        <div className="grow">
          <div className="text-14 font-semibold">Create workspace</div>
          <div className="py-2 text-13 font-medium text-tertiary">
            Instance setup done! Welcome to Vector instance portal. Start your journey with by creating your first
            workspace.
          </div>
          <div className="flex items-center gap-4 pt-2">
            <Link to="/workspace/create" className={getButtonStyling("primary", "lg")}>
              Create workspace
            </Link>
            <Button variant="secondary" size="lg" onClick={() => setDismissed(true)}>
              Close
            </Button>
          </div>
        </div>
        <div className="flex shrink-0 items-center justify-center">
          <img
            src={resolveGeneralTheme(resolvedTheme) === "dark" ? TakeoffIconDark : TakeoffIconLight}
            height={80}
            width={80}
            alt="Vector icon"
          />
        </div>
      </div>
    </div>
  );
};
