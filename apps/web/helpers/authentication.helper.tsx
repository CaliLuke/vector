/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import type { TTranslationStore } from "@plane/i18n";
import { Link } from "@tanstack/react-router";
import { SUPPORT_EMAIL, EAuthErrorCodes as EAuthenticationErrorCodes } from "@plane/constants";
import { createAuthErrorHandler } from "@plane/utils/auth";
export { EAuthErrorCodes as EAuthenticationErrorCodes, EErrorAlertType } from "@plane/constants";
export type { TAuthErrorInfo } from "@plane/constants";

export enum EPageTypes {
  PUBLIC = "PUBLIC",
  NON_AUTHENTICATED = "NON_AUTHENTICATED",
  SET_PASSWORD = "SET_PASSWORD",
  ONBOARDING = "ONBOARDING",
  AUTHENTICATED = "AUTHENTICATED",
}

export { EAuthModes, EAuthSteps } from "@plane/constants";

export const authErrorHandler = createAuthErrorHandler(
  {
    [EAuthenticationErrorCodes.MAGIC_LINK_LOGIN_DISABLED]: {
      title: `Magic link login disabled`,
      message: () => `Magic link login disabled. Please contact your administrator.`,
    },
    [EAuthenticationErrorCodes.PASSWORD_LOGIN_DISABLED]: {
      title: `Password login disabled`,
      message: () => `Password login disabled. Please contact your administrator.`,
    },
    [EAuthenticationErrorCodes.USER_ACCOUNT_DEACTIVATED]: {
      title: `User account deactivated`,
      message: () => `User account deactivated. Please contact ${SUPPORT_EMAIL ? SUPPORT_EMAIL : "administrator"}.`,
    },
    [EAuthenticationErrorCodes.USER_ALREADY_EXIST]: {
      title: `User already exists`,
      message: (email?: string) => (
        <div>
          Your account is already registered.&nbsp;
          <Link
            className="font-medium underline underline-offset-4 transition-all hover:font-bold"
            to="/"
            search={{ email: email || undefined }}
          >
            Sign In
          </Link>
          &nbsp;now.
        </div>
      ),
    },
    [EAuthenticationErrorCodes.USER_DOES_NOT_EXIST]: {
      title: `User does not exist`,
      message: (email?: string) => (
        <div>
          No account found.&nbsp;
          <Link
            className="font-medium underline underline-offset-4 transition-all hover:font-bold"
            to="/"
            search={{ email: email || undefined }}
          >
            Create one
          </Link>
          &nbsp;to get started.
        </div>
      ),
    },
    [EAuthenticationErrorCodes.INVALID_PASSWORD_TOKEN]: {
      title: `Invalid password token`,
      message: () => `Invalid password token.`,
    },
    [EAuthenticationErrorCodes.ADMIN_USER_ALREADY_EXIST]: {
      title: `Admin user already exists`,
      message: () => (
        <div>
          Admin user already exists.&nbsp;
          <Link
            className="font-medium underline underline-offset-4 transition-all hover:font-bold"
            to="/$workspaceSlug/"
            params={{ workspaceSlug: "admin" }}
          >
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
          <Link
            className="font-medium underline underline-offset-4 transition-all hover:font-bold"
            to="/$workspaceSlug/"
            params={{ workspaceSlug: "admin" }}
          >
            Sign In
          </Link>
          &nbsp;now.
        </div>
      ),
    },
    [EAuthenticationErrorCodes.ADMIN_USER_DEACTIVATED]: {
      title: `Admin user deactivated`,
      message: () => <div>Your account is deactivated</div>,
    },
    [EAuthenticationErrorCodes.RATE_LIMIT_EXCEEDED]: {
      title: "",
      message: () => `Rate limit exceeded. Please try again later.`,
    },
  },
  new Set(Object.values(EAuthenticationErrorCodes))
);

export const passwordErrors = [
  EAuthenticationErrorCodes.PASSWORD_TOO_WEAK,
  EAuthenticationErrorCodes.INVALID_NEW_PASSWORD,
];

export function showAuthenticationError(t: TTranslationStore["t"]) {
  setToast({
    type: TOAST_TYPE.ERROR,
    title: t("common.errors.default.title"),
    message: t("common.errors.default.message"),
  });
}
