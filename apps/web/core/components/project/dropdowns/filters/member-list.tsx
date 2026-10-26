/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

// plane imports
import { Button } from "@plane/propel/button";
import { ChevronDownIcon } from "@plane/propel/icons";
import { EUserProjectRoles, EUserWorkspaceRoles } from "@plane/types";
// plane ui
import { CustomMenu } from "@plane/ui";
// components
import { FilterHeader } from "@/components/issues/issue-layouts/filters/header/helpers/filter-header";
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";

interface IRoleOption {
  value: string;
  label: string;
}

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (role: string) => void;
  memberType: "project" | "workspace";
};

const PROJECT_ROLE_OPTIONS: IRoleOption[] = [
  { value: String(EUserProjectRoles.ADMIN), label: "Admin" },
  { value: String(EUserProjectRoles.MEMBER), label: "Member" },
  { value: String(EUserProjectRoles.GUEST), label: "Guest" },
];

const WORKSPACE_ROLE_OPTIONS: IRoleOption[] = [
  { value: String(EUserWorkspaceRoles.ADMIN), label: "Admin" },
  { value: String(EUserWorkspaceRoles.MEMBER), label: "Member" },
  { value: String(EUserWorkspaceRoles.GUEST), label: "Guest" },
  { value: "suspended", label: "Suspended" },
];

// Role filter group component
const RoleFilterGroup = function RoleFilterGroup({
  appliedFilters,
  handleUpdate,
  memberType,
}: {
  appliedFilters: string[] | null;
  handleUpdate: (role: string) => void;
  memberType: "project" | "workspace";
}) {
  const [isExpanded, setIsExpanded] = useState(true);
  const appliedFiltersCount = appliedFilters?.length ?? 0;
  const roleOptions = memberType === "project" ? PROJECT_ROLE_OPTIONS : WORKSPACE_ROLE_OPTIONS;

  return (
    <div className="space-y-2">
      <FilterHeader
        title={`Roles${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
        isPreviewEnabled={isExpanded}
        handleIsPreviewEnabled={() => setIsExpanded(!isExpanded)}
      />

      {isExpanded && (
        <div className="space-y-1">
          {roleOptions.map((role) => {
            const isSelected = appliedFilters?.includes(role.value) ?? false;
            return (
              <FilterOption
                key={`role-${role.value}`}
                isChecked={isSelected}
                title={role.label}
                onClick={() => handleUpdate(role.value)}
              />
            );
          })}
        </div>
      )}
    </div>
  );
};

export const MemberListFilters = function MemberListFilters(props: Props) {
  const { appliedFilters, handleUpdate, memberType } = props;

  return (
    <div className="space-y-4">
      {/* Role Filter Group */}
      <RoleFilterGroup appliedFilters={appliedFilters} handleUpdate={handleUpdate} memberType={memberType} />
    </div>
  );
};

// Dropdown component for member list filters
export const MemberListFiltersDropdown = function MemberListFiltersDropdown(props: Props) {
  const { appliedFilters, handleUpdate, memberType } = props;

  const appliedFiltersCount = appliedFilters?.length ?? 0;

  return (
    <CustomMenu
      customButtonAsChild
      customButton={
        <Button variant="secondary" size="lg" className="relative flex items-center gap-2">
          <span>Filters</span>
          <ChevronDownIcon className="h-3 w-3" />
          {appliedFiltersCount > 0 && (
            <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-accent-primary" />
          )}
        </Button>
      }
      placement="bottom-start"
    >
      <MemberListFilters appliedFilters={appliedFilters} handleUpdate={handleUpdate} memberType={memberType} />
    </CustomMenu>
  );
};
