/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { instanceInfoOptions } from "@/queries/session";
import { useState } from "react";
import { useSearchParams } from "react-router";
// plane imports
import { SitesAuthService } from "@plane/services";
import type { IEmailCheckData } from "@plane/types";
import { OAuthOptions } from "@plane/ui";
// helpers
import type { TAuthErrorInfo } from "@/helpers/authentication.helper";
import { EErrorAlertType, authErrorHandler, EAuthenticationErrorCodes } from "@/helpers/authentication.helper";
// hooks
import { useOAuthConfig } from "@/hooks/oauth";
// types
import { EAuthModes, EAuthSteps } from "@/types/auth";
// local imports
import { TermsAndConditions } from "../terms-and-conditions";
import { AuthBanner } from "./auth-banner";
import { AuthHeader } from "./auth-header";
import { EmailEntryForm } from "@plane/ui";
import { AuthPasswordForm } from "./password";
import { AuthUniqueCodeForm } from "./unique-code";

const authService = new SitesAuthService();

const errorFlow: Partial<Record<EAuthenticationErrorCodes, { mode: EAuthModes; step: EAuthSteps }>> = {
  [EAuthenticationErrorCodes.AUTHENTICATION_FAILED_SIGN_IN]: { mode: EAuthModes.SIGN_IN, step: EAuthSteps.PASSWORD },
  [EAuthenticationErrorCodes.AUTHENTICATION_FAILED_SIGN_UP]: { mode: EAuthModes.SIGN_UP, step: EAuthSteps.PASSWORD },
  [EAuthenticationErrorCodes.INVALID_MAGIC_CODE_SIGN_IN]: { mode: EAuthModes.SIGN_IN, step: EAuthSteps.UNIQUE_CODE },
  [EAuthenticationErrorCodes.EXPIRED_MAGIC_CODE_SIGN_IN]: { mode: EAuthModes.SIGN_IN, step: EAuthSteps.UNIQUE_CODE },
  [EAuthenticationErrorCodes.EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_IN]: {
    mode: EAuthModes.SIGN_IN,
    step: EAuthSteps.UNIQUE_CODE,
  },
  [EAuthenticationErrorCodes.INVALID_MAGIC_CODE_SIGN_UP]: { mode: EAuthModes.SIGN_UP, step: EAuthSteps.UNIQUE_CODE },
  [EAuthenticationErrorCodes.EXPIRED_MAGIC_CODE_SIGN_UP]: { mode: EAuthModes.SIGN_UP, step: EAuthSteps.UNIQUE_CODE },
  [EAuthenticationErrorCodes.EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_UP]: {
    mode: EAuthModes.SIGN_UP,
    step: EAuthSteps.UNIQUE_CODE,
  },
};
function initialAuthState(errorCode: string) {
  const error = authErrorHandler(errorCode as EAuthenticationErrorCodes);
  const flow = error && errorFlow[error.code];
  return { error, mode: flow?.mode ?? EAuthModes.SIGN_UP, step: flow?.step ?? EAuthSteps.EMAIL };
}
export function AuthRoot() {
  const [params] = useSearchParams();
  const email = params.get("email") ?? "";
  const error = params.get("error_code") ?? "";
  return (
    <AuthFlow
      key={JSON.stringify([email, error])}
      emailParam={email}
      errorCode={error}
      nextPath={params.get("next_path") ?? undefined}
    />
  );
}
function AuthFlow({
  emailParam,
  errorCode,
  nextPath,
}: {
  emailParam: string;
  errorCode: string;
  nextPath: string | undefined;
}) {
  const { error: initialError, mode: initialMode, step: initialStep } = initialAuthState(errorCode);
  // states
  const [authMode, setAuthMode] = useState<EAuthModes>(initialMode);
  const [authStep, setAuthStep] = useState<EAuthSteps>(initialStep);
  const [email, setEmail] = useState(emailParam);
  const [errorInfo, setErrorInfo] = useState<TAuthErrorInfo | undefined>(initialError);
  // hooks
  const { data: instanceInfo } = useQuery(instanceInfoOptions);
  const {
    is_smtp_configured: isSMTPConfigured = false,
    is_magic_login_enabled: isMagicLoginEnabled = false,
    is_email_password_enabled: isEmailPasswordEnabled = false,
  } = instanceInfo?.config ?? {};
  const oAuthActionText = authMode === EAuthModes.SIGN_UP ? "Sign up" : "Sign in";
  const { isOAuthEnabled, oAuthOptions } = useOAuthConfig(oAuthActionText);

  // submit handler- email verification
  const handleEmailVerification = async (data: IEmailCheckData) => {
    setEmail(data.email);

    await authService
      .emailCheck(data)
      .then(async (response) => {
        const currentAuthMode = response.existing ? EAuthModes.SIGN_IN : EAuthModes.SIGN_UP;
        setAuthMode(currentAuthMode);
        if (isSMTPConfigured && isMagicLoginEnabled && response.status === "MAGIC_CODE") {
          setAuthStep(EAuthSteps.UNIQUE_CODE);
          await generateEmailUniqueCode(data.email);
        } else if (isEmailPasswordEnabled) {
          setAuthStep(EAuthSteps.PASSWORD);
        } else {
          setErrorInfo({
            type: EErrorAlertType.BANNER_ALERT,
            code: EAuthenticationErrorCodes.AUTHENTICATION_FAILED_SIGN_IN,
            title: "Email authentication is unavailable",
            message: "Use another sign-in method or contact your administrator.",
          });
        }
        return;
      })
      .catch((error) => {
        const errorhandler = authErrorHandler(error?.error_code?.toString(), data?.email || undefined);
        if (errorhandler?.type) setErrorInfo(errorhandler);
      });
  };

  // generating the unique code
  const generateEmailUniqueCode = async (recipient: string): Promise<void> => {
    const payload = { email: recipient };
    return await authService.generateUniqueCode(payload).catch((error) => {
      const errorhandler = authErrorHandler(error?.error_code.toString());
      if (errorhandler?.type) setErrorInfo(errorhandler);
      throw error;
    });
  };

  return (
    <div className="mt-10 flex w-full flex-grow flex-col items-center justify-center py-6">
      <div className="relative flex w-full max-w-[22.5rem] flex-col gap-6">
        {errorInfo && errorInfo?.type === EErrorAlertType.BANNER_ALERT && (
          <AuthBanner bannerData={errorInfo} handleBannerData={(value) => setErrorInfo(value)} />
        )}
        <AuthHeader authMode={authMode} />
        {isOAuthEnabled && <OAuthOptions options={oAuthOptions} compact={authStep === EAuthSteps.PASSWORD} />}

        {authStep === EAuthSteps.EMAIL && (
          <EmailEntryForm className="mt-5" defaultEmail={email} onSubmit={handleEmailVerification} />
        )}
        {authStep === EAuthSteps.UNIQUE_CODE && (
          <AuthUniqueCodeForm
            mode={authMode}
            email={email}
            nextPath={nextPath}
            handleEmailClear={() => {
              setEmail("");
              setAuthStep(EAuthSteps.EMAIL);
            }}
            generateEmailUniqueCode={generateEmailUniqueCode}
          />
        )}
        {authStep === EAuthSteps.PASSWORD && (
          <AuthPasswordForm
            mode={authMode}
            isSMTPConfigured={isSMTPConfigured}
            email={email}
            nextPath={nextPath}
            handleEmailClear={() => {
              setEmail("");
              setAuthStep(EAuthSteps.EMAIL);
            }}
            handleAuthStep={(step: EAuthSteps) => {
              if (step === EAuthSteps.UNIQUE_CODE) generateEmailUniqueCode(email);
              setAuthStep(step);
            }}
          />
        )}
        <TermsAndConditions isSignUp={authMode === EAuthModes.SIGN_UP} />
      </div>
    </div>
  );
}
