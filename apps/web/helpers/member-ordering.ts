/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Types and utilities for member filtering
import type { TMemberOrderByOptions } from "@plane/constants";
import type { IUserLite, TProjectMembership } from "@plane/types";
import lodashOrderBy from "lodash-es/orderBy.js";

export interface IMemberFilters {
  order_by?: TMemberOrderByOptions;
  roles?: string[];
}

// Helper function to parse order key and direction
const parseOrderKey = (orderKey?: TMemberOrderByOptions): { field: string; direction: "asc" | "desc" } => {
  // Default to sorting by display_name in ascending order when no order key is provided
  if (!orderKey) {
    return {
      field: "display_name",
      direction: "asc",
    };
  }

  const isDescending = orderKey.startsWith("-");
  const field = isDescending ? orderKey.slice(1) : orderKey;
  return {
    field,
    direction: isDescending ? "desc" : "asc",
  };
};

// Unified function to get sort key for any member type
const getMemberSortKey = (memberDetails: IUserLite, field: string, memberRole?: string): string | Date => {
  switch (field) {
    case "display_name":
      return memberDetails.display_name?.toLowerCase() || "";
    case "full_name": {
      const firstName = memberDetails.first_name || "";
      const lastName = memberDetails.last_name || "";
      return `${firstName} ${lastName}`.toLowerCase().trim();
    }
    case "email":
      return memberDetails.email?.toLowerCase() || "";
    case "joining_date": {
      if (!memberDetails.joining_date) {
        // Return a very old date for missing dates to sort them last
        return new Date(0);
      }
      const date = new Date(memberDetails.joining_date);
      // Return a very old date for invalid dates to sort them last
      return isNaN(date.getTime()) ? new Date(0) : date;
    }
    case "role":
      return (memberRole ?? "").toLowerCase();
    default:
      return "";
  }
};

// Filter functions
const filterProjectMembersByRole = (members: TProjectMembership[], roleFilters: string[]): TProjectMembership[] => {
  if (roleFilters.length === 0) return members;

  return members.filter((member) => {
    const memberRole = String(member.role ?? member.original_role ?? "");
    return roleFilters.includes(memberRole);
  });
};

const filterWorkspaceMembersByRole = <T extends { role: string | number; is_active?: boolean }>(
  members: T[],
  roleFilters: string[]
): T[] => {
  if (roleFilters.length === 0) return members;

  return members.filter((member) => {
    const memberRole = String(member.role ?? "");
    const isSuspended = member.is_active === false;

    // Check if suspended is in the role filters
    const hasSuspendedFilter = roleFilters.includes("suspended");
    // Get non-suspended role filters
    const activeRoleFilters = roleFilters.filter((role) => role !== "suspended");

    // For suspended users, include them only if suspended filter is selected
    if (isSuspended) {
      return hasSuspendedFilter;
    }

    // For active users, include them only if their role matches any active role filter
    return activeRoleFilters.includes(memberRole);
  });
};

// Unified sorting function
const sortMembers = <T>(
  members: T[],
  memberDetailsMap: Record<string, IUserLite>,
  getMemberKey: (member: T) => string,
  getMemberRole: (member: T) => string,
  orderBy?: TMemberOrderByOptions
): T[] => {
  if (!orderBy) return members;

  const { field, direction } = parseOrderKey(orderBy);

  return lodashOrderBy(
    members,
    [
      (member) => (memberDetailsMap[getMemberKey(member)] ? 0 : 1),
      (member) => {
        const details = memberDetailsMap[getMemberKey(member)];
        return details ? getMemberSortKey(details, field, getMemberRole(member)) : "";
      },
    ],
    ["asc", direction]
  );
};

// Specific implementations using the unified functions
export const sortProjectMembers = (
  members: TProjectMembership[],
  memberDetailsMap: Record<string, IUserLite>,
  getMemberKey: (member: TProjectMembership) => string,
  filters?: IMemberFilters
): TProjectMembership[] => {
  // Apply role filtering first
  const filteredMembers =
    filters?.roles && filters.roles.length > 0 ? filterProjectMembersByRole(members, filters.roles) : members;

  // If no order_by filter, return filtered members
  if (!filters?.order_by) return filteredMembers;

  // Apply sorting
  return sortMembers(
    filteredMembers,
    memberDetailsMap,
    getMemberKey,
    (member) => String(member.role ?? member.original_role ?? ""),
    filters.order_by
  );
};

export const sortWorkspaceMembers = <T extends { role: string | number; is_active?: boolean }>(
  members: T[],
  memberDetailsMap: Record<string, IUserLite>,
  getMemberKey: (member: T) => string,
  filters?: IMemberFilters
): T[] => {
  const filteredMembers =
    filters?.roles && filters.roles.length > 0 ? filterWorkspaceMembersByRole(members, filters.roles) : members;

  // If no order_by filter, return filtered members
  if (!filters?.order_by) return filteredMembers;

  // Apply sorting
  return sortMembers(
    filteredMembers,
    memberDetailsMap,
    getMemberKey,
    (member) => String(member.role ?? ""),
    filters.order_by
  );
};
