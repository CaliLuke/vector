/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery, skipToken } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import type { IWorkspaceMemberInvitation } from "@plane/types";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { WorkspaceLogo } from "@/components/workspace/logo";
// helpers
import { EAuthModes, EAuthSteps } from "@/helpers/authentication.helper";
// services
import { WorkspaceService } from "@/services/workspace.service";

type TAuthHeader = {
  workspaceSlug: string | undefined;
  invitationId: string | undefined;
  invitationEmail: string | undefined;
  authMode: EAuthModes;
  currentAuthStep: EAuthSteps;
};

const Titles = {
  [EAuthModes.SIGN_IN]: {
    [EAuthSteps.EMAIL]: {
      header: "Work in all dimensions.",
      subHeader: "Welcome back to Vector.",
    },
    [EAuthSteps.PASSWORD]: {
      header: "Work in all dimensions.",
      subHeader: "Welcome back to Vector.",
    },
    [EAuthSteps.UNIQUE_CODE]: {
      header: "Work in all dimensions.",
      subHeader: "Welcome back to Vector.",
    },
  },
  [EAuthModes.SIGN_UP]: {
    [EAuthSteps.EMAIL]: {
      header: "Work in all dimensions.",
      subHeader: "Create your Vector account.",
    },
    [EAuthSteps.PASSWORD]: {
      header: "Work in all dimensions.",
      subHeader: "Create your Vector account.",
    },
    [EAuthSteps.UNIQUE_CODE]: {
      header: "Work in all dimensions.",
      subHeader: "Create your Vector account.",
    },
  },
};

const workSpaceService = new WorkspaceService();

export const AuthHeader = function AuthHeader(props: TAuthHeader) {
  const { workspaceSlug, invitationId, invitationEmail, authMode, currentAuthStep } = props;
  // plane imports
  const { t } = useTranslation();

  const { data: invitation, isLoading } = useQuery({
    queryKey: ["WORKSPACE_INVITATION", workspaceSlug, invitationId],
    queryFn:
      workspaceSlug && invitationId
        ? async () =>
            workspaceSlug && invitationId && workSpaceService.getWorkspaceInvitation(workspaceSlug, invitationId)
        : skipToken,
    refetchOnWindowFocus: false,
    retry: false,
  });

  const getHeaderSubHeader = (
    step: EAuthSteps,
    mode: EAuthModes,
    invitation: IWorkspaceMemberInvitation | undefined,
    email: string | undefined
  ) => {
    if (invitation && email && invitation.email === email && invitation.workspace) {
      const workspace = invitation.workspace;
      return {
        header: (
          <div className="relative inline-flex items-center gap-2">
            {t("common.join")}{" "}
            <WorkspaceLogo logo={workspace?.logo_url} name={workspace?.name} classNames="size-9 flex-shrink-0" />{" "}
            {workspace.name}
          </div>
        ),
        subHeader:
          mode == EAuthModes.SIGN_UP
            ? "Create an account to start managing work with your team."
            : "Log in to start managing work with your team.",
      };
    }

    return Titles[mode][step];
  };

  const { header, subHeader } = getHeaderSubHeader(currentAuthStep, authMode, invitation || undefined, invitationEmail);

  if (isLoading)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );

  return <AuthHeaderBase subHeader={subHeader} header={header} />;
};

type TAuthHeaderBase = {
  header: React.ReactNode;
  subHeader: string;
};

export function AuthHeaderBase(props: TAuthHeaderBase) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-h4-semibold text-primary">{props.header}</span>
      <span className="text-h4-semibold text-placeholder">{props.subHeader}</span>
    </div>
  );
}
