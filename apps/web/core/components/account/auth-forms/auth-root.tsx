/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation } from "@tanstack/react-router";
import { useState } from "react";

// plane imports
import { OAuthOptions } from "@plane/ui";
// helpers
import type { TAuthErrorInfo } from "@/helpers/authentication.helper";
import {
  EAuthModes,
  EAuthSteps,
  EAuthenticationErrorCodes,
  EErrorAlertType,
  authErrorHandler,
} from "@/helpers/authentication.helper";
// hooks
import { useOAuthConfig } from "@/hooks/oauth";
import { useQuery } from "@tanstack/react-query";
import { instanceConfigOptions } from "@/queries/instance";
// local imports
import { TermsAndConditions } from "../terms-and-conditions";
import { AuthBanner } from "./auth-banner";
import { AuthHeader, AuthHeaderBase } from "./auth-header";
import { AuthFormRoot } from "./form-root";

type TAuthRoot = {
  authMode: EAuthModes;
};

const errorFlows = [
  {
    mode: EAuthModes.SIGN_UP,
    step: EAuthSteps.PASSWORD,
    codes: [EAuthenticationErrorCodes.AUTHENTICATION_FAILED_SIGN_UP],
  },
  {
    mode: EAuthModes.SIGN_IN,
    step: EAuthSteps.PASSWORD,
    codes: [EAuthenticationErrorCodes.AUTHENTICATION_FAILED_SIGN_IN],
  },
  {
    mode: EAuthModes.SIGN_UP,
    step: EAuthSteps.UNIQUE_CODE,
    codes: [
      EAuthenticationErrorCodes.INVALID_MAGIC_CODE_SIGN_UP,
      EAuthenticationErrorCodes.INVALID_EMAIL_MAGIC_SIGN_UP,
      EAuthenticationErrorCodes.EXPIRED_MAGIC_CODE_SIGN_UP,
      EAuthenticationErrorCodes.EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_UP,
    ],
  },
  {
    mode: EAuthModes.SIGN_IN,
    step: EAuthSteps.UNIQUE_CODE,
    codes: [
      EAuthenticationErrorCodes.INVALID_MAGIC_CODE_SIGN_IN,
      EAuthenticationErrorCodes.INVALID_EMAIL_MAGIC_SIGN_IN,
      EAuthenticationErrorCodes.EXPIRED_MAGIC_CODE_SIGN_IN,
      EAuthenticationErrorCodes.EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_IN,
    ],
  },
];

function initialAuthentication(errorCode: string | null, mode: EAuthModes) {
  const error = authErrorHandler(errorCode as EAuthenticationErrorCodes);
  const flow = errorFlows.find((candidate) => error && candidate.codes.includes(error.code));
  return { error, mode: flow?.mode ?? mode, step: flow?.step ?? EAuthSteps.EMAIL };
}

export function AuthRoot({ authMode }: TAuthRoot) {
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const scope = [
    authMode,
    searchParams.get("email"),
    searchParams.get("error_code"),
    searchParams.get("slug"),
    searchParams.get("invitation_id"),
  ];
  return <AuthFlow key={JSON.stringify(scope)} authMode={authMode} searchParams={searchParams} />;
}

function AuthFlow({ authMode: currentAuthMode, searchParams }: TAuthRoot & { searchParams: URLSearchParams }) {
  // query params
  const emailParam = searchParams.get("email");
  const invitation_id = searchParams.get("invitation_id");
  const workspaceSlug = searchParams.get("slug");
  const error_code = searchParams.get("error_code");
  // states
  const initial = initialAuthentication(error_code, currentAuthMode);
  const [authMode, setAuthMode] = useState<EAuthModes>(initial.mode);
  const [authStep, setAuthStep] = useState<EAuthSteps>(initial.step);
  const [email, setEmail] = useState(emailParam ?? "");
  const [errorInfo, setErrorInfo] = useState<TAuthErrorInfo | undefined>(initial.error);
  // store hooks
  const { data: config } = useQuery(instanceConfigOptions);
  // derived values
  const oAuthActionText = authMode === EAuthModes.SIGN_UP ? "Sign up" : "Sign in";
  const { isOAuthEnabled, oAuthOptions } = useOAuthConfig(oAuthActionText);
  const isEmailBasedAuthEnabled = config?.is_email_password_enabled || config?.is_magic_login_enabled;
  const noAuthMethodsAvailable = !isOAuthEnabled && !isEmailBasedAuthEnabled;

  if (noAuthMethodsAvailable) {
    return (
      <AuthContainer>
        <AuthHeaderBase
          header="No authentication methods available"
          subHeader="Please contact your administrator to enable authentication for your instance."
        />
      </AuthContainer>
    );
  }

  return (
    <AuthContainer>
      {errorInfo?.type === EErrorAlertType.BANNER_ALERT && (
        <AuthBanner message={errorInfo.message} handleBannerData={(value) => setErrorInfo(value)} />
      )}
      <AuthHeader
        workspaceSlug={workspaceSlug ?? undefined}
        invitationId={invitation_id ?? undefined}
        invitationEmail={email || undefined}
        authMode={authMode}
        currentAuthStep={authStep}
      />
      {isOAuthEnabled && (
        <OAuthOptions
          options={oAuthOptions}
          compact={authStep === EAuthSteps.PASSWORD}
          showDivider={isEmailBasedAuthEnabled}
        />
      )}
      {isEmailBasedAuthEnabled && (
        <AuthFormRoot
          authStep={authStep}
          authMode={authMode}
          email={email}
          setEmail={(email) => setEmail(email)}
          setAuthMode={(authMode) => setAuthMode(authMode)}
          setAuthStep={(authStep) => setAuthStep(authStep)}
          setErrorInfo={(errorInfo) => setErrorInfo(errorInfo)}
          currentAuthMode={currentAuthMode}
        />
      )}
      <TermsAndConditions authType={authMode} />
    </AuthContainer>
  );
}

function AuthContainer({ children }: { children: React.ReactNode }) {
  return (
    <div className="mt-10 flex w-full flex-grow flex-col items-center justify-center py-6">
      <div className="relative flex w-full max-w-[22.5rem] flex-col gap-6">{children}</div>
    </div>
  );
}
