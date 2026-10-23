/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import React from "react";
// components
import { AuthBase } from "@/components/auth-screens/auth-base";
// helpers
import { EAuthModes, EPageTypes } from "@/helpers/authentication.helper";
// layouts
import DefaultLayout from "@/layouts/default-layout";
// wrappers
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

function HomePage(): React.JSX.Element | null {
  return (
    <DefaultLayout>
      <AuthenticationWrapper pageType={EPageTypes.NON_AUTHENTICATED}>
        <AuthBase authType={EAuthModes.SIGN_IN} />
      </AuthenticationWrapper>
    </DefaultLayout>
  );
}

export const Route = createFileRoute("/_home/")({
  component: HomePage,
});
