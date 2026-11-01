/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EAuthModes } from "@plane/constants";
import { SetPasswordForm } from "@/components/account/auth-forms/set-password";
import { ResetPasswordForm } from "@/components/account/auth-forms/reset-password";
import { AuthHeader } from "@/components/auth-screens/header";
import { EPageTypes } from "@/helpers/authentication.helper";
import DefaultLayout from "@/layouts/default-layout";
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

export function PasswordScreen({
  pageType,
}: {
  pageType: EPageTypes.NON_AUTHENTICATED | EPageTypes.SET_PASSWORD;
}): React.JSX.Element | null {
  return (
    <DefaultLayout>
      <AuthenticationWrapper pageType={pageType}>
        <div className="relative z-10 flex h-screen w-screen flex-col items-center overflow-hidden overflow-y-auto px-8 pt-6 pb-10">
          <AuthHeader type={EAuthModes.SIGN_IN} />
          {pageType === EPageTypes.SET_PASSWORD ? <SetPasswordForm /> : <ResetPasswordForm />}
        </div>
      </AuthenticationWrapper>
    </DefaultLayout>
  );
}
