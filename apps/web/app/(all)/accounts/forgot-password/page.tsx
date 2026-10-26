/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

// components
import { ForgotPasswordForm } from "@/components/account/auth-forms/forgot-password";
import { AuthHeader } from "@/components/auth-screens/header";
// helpers
import { EAuthModes, EPageTypes } from "@/helpers/authentication.helper";
// layouts
import DefaultLayout from "@/layouts/default-layout";
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

function ForgotPasswordPage(): React.JSX.Element | null {
  return (
    <DefaultLayout>
      <AuthenticationWrapper pageType={EPageTypes.NON_AUTHENTICATED}>
        <div className="relative z-10 flex h-screen w-screen flex-col items-center overflow-hidden overflow-y-auto px-8 pt-6 pb-10">
          <AuthHeader type={EAuthModes.SIGN_IN} />
          <ForgotPasswordForm />
        </div>
      </AuthenticationWrapper>
    </DefaultLayout>
  );
}

export const Route = createFileRoute("/accounts/forgot-password/")({
  component: ForgotPasswordPage,
});
