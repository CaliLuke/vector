/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute } from "@tanstack/react-router";
import { AuthBase } from "@/components/auth-screens/auth-base";
// helpers
import { EAuthModes, EPageTypes } from "@/helpers/authentication.helper";
// assets
import DefaultLayout from "@/layouts/default-layout";
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

function SignUpPage(): React.JSX.Element | null {
  return (
    <DefaultLayout>
      <AuthenticationWrapper pageType={EPageTypes.NON_AUTHENTICATED}>
        <AuthBase authType={EAuthModes.SIGN_UP} />
      </AuthenticationWrapper>
    </DefaultLayout>
  );
}

export const Route = createFileRoute("/sign-up/")({
  component: SignUpPage,
});
