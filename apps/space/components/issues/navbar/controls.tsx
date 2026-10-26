/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { IssueFiltersDropdown } from "@/components/issues/filters";
// helpers
// hooks
import useIsInIframe from "@/hooks/use-is-in-iframe";
import type { TProjectPublishSettings } from "@plane/types";
// types
// local imports
import { IssuesLayoutSelection } from "./layout-selection";
import { NavbarTheme } from "./theme";
import { UserAvatar } from "./user-avatar";

export type NavbarControlsProps = {
  publishSettings: TProjectPublishSettings;
};

export const NavbarControls = function NavbarControls(props: NavbarControlsProps) {
  // props
  const { publishSettings } = props;
  const { anchor } = publishSettings;
  const isInIframe = useIsInIframe();

  if (!anchor) return null;

  return (
    <>
      {/* issue views */}
      <div className="shrink-0">
        <IssuesLayoutSelection anchor={anchor} />
      </div>

      {/* issue filters */}
      <div className="shrink-0">
        <IssueFiltersDropdown anchor={anchor} />
      </div>

      {/* theming */}
      <div className="shrink-0">
        <NavbarTheme />
      </div>

      {!isInIframe && <UserAvatar />}
    </>
  );
};
