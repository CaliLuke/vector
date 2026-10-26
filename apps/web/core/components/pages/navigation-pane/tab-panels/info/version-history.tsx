/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { usePageRecord } from "@/components/pages/page-record-context";
import { projectPageVersionsOptions } from "@/queries/pages";

import type { TPage } from "@plane/types";

import { Link, useLocation } from "@tanstack/react-router";
import { useCallback } from "react";

import { useQuery, useQueryClient } from "@tanstack/react-query";
// plane imports
import { useTranslation } from "@plane/i18n";
import type { IUserLite, TPageVersion } from "@plane/types";
import { Avatar } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
import { renderFormattedDate, renderFormattedTime } from "@plane/utils/datetime";
// components
// hooks
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useQueryParams } from "@/hooks/use-query-params";
// store
// local imports
import { PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM } from "../..";

type Props = {
  page: TPage;
  workspaceSlug: string;
};

type VersionHistoryItemProps = {
  getVersionLink: (versionID: string) => string;
  isVersionActive: boolean;
  memberDetails: IUserLite | undefined;
  version: TPageVersion;
};

function VersionHistoryItem(props: VersionHistoryItemProps) {
  const { getVersionLink, isVersionActive, memberDetails, version } = props;
  // translation
  const { t } = useTranslation();

  return (
    <li className="relative flex items-center gap-x-4 text-11 font-medium">
      {/* timeline icon */}
      <div className="relative grid size-6 flex-none place-items-center">
        <div className="size-2 rounded-full bg-layer-3" />
      </div>
      {/* end timeline icon */}
      <Link
        to={getVersionLink(version.id)}
        className={cn("block flex-1 rounded-md px-1 py-2 hover:bg-layer-transparent-hover", {
          "bg-layer-transparent-selected hover:bg-layer-transparent-selected": isVersionActive,
        })}
      >
        <p className="text-tertiary">
          {renderFormattedDate(version.last_saved_at)}, {renderFormattedTime(version.last_saved_at)}
        </p>
        <p className="mt-1 flex items-center gap-1">
          <Avatar
            size="sm"
            src={getFileURL(memberDetails?.avatar_url ?? "")}
            name={memberDetails?.display_name}
            className="shrink-0"
          />
          <span>{memberDetails?.display_name ?? t("common.deactivated_user")}</span>
        </p>
      </Link>
    </li>
  );
}

export function PageNavigationPaneInfoTabVersionHistory(props: Props) {
  const { page, workspaceSlug } = props;
  // navigation
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const activeVersion = searchParams.get(PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM);
  // derived values
  const { id } = page;
  // translation
  const { t } = useTranslation();
  // query params
  const { updateQueryParams } = useQueryParams();
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  // fetch all versions
  const { scope } = usePageRecord();
  const client = useQueryClient();
  const { data: versionsList } = useQuery(projectPageVersionsOptions(client, scope, id));

  const getVersionLink = useCallback(
    (versionID?: string) => {
      if (versionID) {
        return updateQueryParams({
          paramsToAdd: { [PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM]: versionID },
        });
      } else {
        return updateQueryParams({
          paramsToRemove: [PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM],
        });
      }
    },
    [updateQueryParams]
  );

  return (
    <div>
      <p className="text-11 font-medium text-secondary">{t("page_navigation_pane.tabs.info.version_history.label")}</p>
      <div className="mt-3">
        <ul className="relative">
          {/* timeline line */}
          <div className={cn("absolute top-0 left-0 flex h-full w-6 justify-center")}>
            <div className="w-px bg-layer-3" />
          </div>
          {/* end timeline line */}
          <li className="relative flex items-center gap-x-4 text-11 font-medium">
            {/* timeline icon */}
            <div className="relative grid size-6 flex-none place-items-center rounded-full bg-accent-primary/20">
              <div className="size-2.5 rounded-full bg-accent-primary/40" />
            </div>
            {/* end timeline icon */}
            <Link
              to={getVersionLink()}
              className={cn("flex-1 rounded-md bg-layer-transparent px-1 py-2 hover:bg-layer-transparent-hover", {
                "bg-layer-transparent-selected hover:bg-layer-transparent-selected": !activeVersion,
              })}
            >
              {t("page_navigation_pane.tabs.info.version_history.current_version")}
            </Link>
          </li>
          {versionsList?.map((version) => (
            <VersionHistoryItem
              key={version.id}
              getVersionLink={getVersionLink}
              isVersionActive={activeVersion === version.id}
              memberDetails={membersById?.get(version.owned_by)?.member}
              version={version}
            />
          ))}
        </ul>
      </div>
    </div>
  );
}
