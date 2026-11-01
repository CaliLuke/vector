/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "react-router";
import { SUPPORT_EMAIL, EAuthErrorCodes as EAuthenticationErrorCodes } from "@plane/constants";
import { createAuthErrorHandler } from "@plane/utils/auth";
export { EAuthErrorCodes as EAuthenticationErrorCodes, EErrorAlertType } from "@plane/constants";
export type { TAuthErrorInfo } from "@plane/constants";

export const authErrorHandler = createAuthErrorHandler({
  [EAuthenticationErrorCodes.USER_ALREADY_EXIST]: {
    title: `User already exists`,
    message: (email?: string) => (
      <div>
        Your account is already registered.&nbsp;
        <Link
          className="font-medium underline underline-offset-4 transition-all hover:font-bold"
          to={`/sign-in${email ? `?email=${encodeURIComponent(email)}` : ``}`}
        >
          Sign In
        </Link>
        &nbsp;now.
      </div>
    ),
  },
  [EAuthenticationErrorCodes.USER_ACCOUNT_DEACTIVATED]: {
    title: `User account deactivated`,
    message: () => `User account deactivated. Please contact ${SUPPORT_EMAIL ? SUPPORT_EMAIL : "administrator"}.`,
  },
  [EAuthenticationErrorCodes.USER_DOES_NOT_EXIST]: {
    title: `User does not exist`,
    message: (email?: string) => (
      <div>
        No account found.&nbsp;
        <Link
          className="font-medium underline underline-offset-4 transition-all hover:font-bold"
          to={`/${email ? `?email=${encodeURIComponent(email)}` : ``}`}
        >
          Create one
        </Link>
        &nbsp;to get started.
      </div>
    ),
  },
  [EAuthenticationErrorCodes.ADMIN_USER_ALREADY_EXIST]: {
    title: `Admin user already exists`,
    message: () => (
      <div>
        Admin user already exists.&nbsp;
        <Link className="font-medium underline underline-offset-4 transition-all hover:font-bold" to={`/admin`}>
          Sign In
        </Link>
        &nbsp;now.
      </div>
    ),
  },
  [EAuthenticationErrorCodes.ADMIN_USER_DOES_NOT_EXIST]: {
    title: `Admin user does not exist`,
    message: () => (
      <div>
        Admin user does not exist.&nbsp;
        <Link className="font-medium underline underline-offset-4 transition-all hover:font-bold" to={`/admin`}>
          Sign In
        </Link>
        &nbsp;now.
      </div>
    ),
  },
});
