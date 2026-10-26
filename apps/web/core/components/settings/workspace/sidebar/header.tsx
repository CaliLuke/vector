/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";

// plane imports
import { ROLE_DETAILS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { IconButton } from "@plane/propel/icon-button";
// components
import { WorkspaceLogo } from "@/components/workspace/logo";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";

import { useCurrentWorkspace } from "@/hooks/use-workspaces";

export const WorkspaceSettingsSidebarHeader = function WorkspaceSettingsSidebarHeader() {
  // router
  const router = useRouter();
  // store hooks
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { workspaceRole: currentWorkspaceRole } = useWorkspacePermissions(workspaceSlug);
  // derived values
  // translation
  const { t } = useTranslation();

  if (!currentWorkspaceRole) return null;

  return (
    <div className="shrink-0">
      <div className="flex items-center gap-1 py-3 pr-5 pl-4 text-body-md-medium">
        <IconButton
          variant="ghost"
          size="base"
          icon={ArrowLeft}
          onClick={() => void router.navigate({ href: `/${currentWorkspace?.slug}/` })}
        />
        <p>Workspace settings</p>
      </div>
      <div className="mt-1.5 flex items-center justify-between gap-2 px-5 py-0.5">
        <div className="flex items-center gap-2 truncate">
          <WorkspaceLogo
            logo={currentWorkspace?.logo_url}
            name={currentWorkspace?.name}
            classNames="shrink-0 size-8 border border-subtle"
          />
          <div className="truncate">
            <p className="truncate text-body-sm-medium">{currentWorkspace?.name}</p>
            <p className="truncate text-caption-md-regular">{t(ROLE_DETAILS[currentWorkspaceRole].i18n_title)}</p>
          </div>
        </div>
      </div>
    </div>
  );
};
