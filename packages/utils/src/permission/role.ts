/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { EUserPermissions } from "@plane/constants";

export const getUserRole = (role: unknown) => {
  switch (role) {
    case EUserPermissions.GUEST:
      return "GUEST";
    case EUserPermissions.MEMBER:
      return "MEMBER";
    case EUserPermissions.ADMIN:
      return "ADMIN";
  }

  return undefined;
};

type TSupportedRole = EUserPermissions;

/**
 * @description Returns the highest role from an array of supported roles
 * @param { TSupportedRole[] } roles
 * @returns { TSupportedRole | undefined }
 */
export const getHighestRole = <T extends TSupportedRole>(roles: T[]): T | undefined => {
  if (!roles || roles.length === 0) return undefined;
  return roles.reduce((highest, current) => (current > highest ? current : highest));
};
