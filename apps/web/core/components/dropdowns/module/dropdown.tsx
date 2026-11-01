/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// hooks
import { useProjectModules } from "@/hooks/use-modules";
// types
// local imports
import { ModuleDropdownBase } from "./base";
import type { TModuleDropdownProps } from "./base";

export const ModuleDropdown = function ModuleDropdown(props: TModuleDropdownProps) {
  const { projectId } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const catalogue = useProjectModules(workspaceSlug, projectId);

  return (
    <ModuleDropdownBase
      {...props}
      modules={catalogue.modules}
      isError={catalogue.isError || catalogue.isAccessDenied}
      retry={catalogue.retry}
    />
  );
};
