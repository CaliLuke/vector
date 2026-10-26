/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import type { ReactNode } from "react";

// hooks
import { useProjectModules } from "@/hooks/use-modules";
// types
import type { TDropdownProps } from "../types";
// local imports
import { ModuleDropdownBase } from "./base";

type TModuleDropdownProps = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  projectId: string | undefined;
  showCount?: boolean;
  onClose?: () => void;
  renderByDefault?: boolean;
  itemClassName?: string;
} & (
    | {
        multiple: false;
        onChange: (val: string | null) => void;
        value: string | null;
      }
    | {
        multiple: true;
        onChange: (val: string[]) => void;
        value: string[] | null;
      }
  );

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
