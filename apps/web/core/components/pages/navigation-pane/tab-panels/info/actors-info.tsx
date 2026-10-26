/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { Link } from "@tanstack/react-router";

// plane imports
import { useTranslation } from "@plane/i18n";
import { Avatar } from "@plane/ui";
import type { IUserLite } from "@plane/types";
import { calculateTimeAgoShort, renderFormattedDate } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
// hooks
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
// store

type Props = {
  page: TPage;
  workspaceSlug: string;
};

export function PageNavigationPaneInfoTabActorsInfo(props: Props) {
  const { page, workspaceSlug } = props;
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  // derived values
  const { created_by, updated_by } = page;
  const editorInformation = memberById(updated_by)?.member;
  const creatorInformation = memberById(created_by)?.member;
  // translation
  const { t } = useTranslation();

  return (
    <div className="mt-4 space-y-3">
      <PageActorRow
        workspaceSlug={workspaceSlug}
        userId={updated_by}
        member={editorInformation}
        label={t("page_navigation_pane.tabs.info.actors_info.edited_by")}
        timestamp={`${calculateTimeAgoShort(page.updated_at ?? "")} ago`}
      />
      <PageActorRow
        workspaceSlug={workspaceSlug}
        userId={created_by}
        member={creatorInformation}
        label={t("page_navigation_pane.tabs.info.actors_info.created_by")}
        timestamp={renderFormattedDate(page.created_at)}
      />
    </div>
  );
}

function PageActorRow({
  workspaceSlug,
  userId,
  member,
  label,
  timestamp,
}: {
  workspaceSlug: string;
  userId: string | undefined;
  member: IUserLite | undefined;
  label: string;
  timestamp: string | undefined;
}) {
  const { t } = useTranslation();
  return (
    <div>
      <p className="text-11 font-medium text-tertiary">{label}</p>
      <div className="mt-2 flex items-center justify-between gap-2 text-13 font-medium">
        <Link
          to="/$workspaceSlug/profile/$userId/"
          params={{ workspaceSlug, userId: userId ?? "" }}
          disabled={!userId}
          className="flex items-center gap-1"
        >
          <Avatar
            src={getFileURL(member?.avatar_url ?? "")}
            name={member?.display_name}
            className="flex-shrink-0"
            size="sm"
          />
          <span>{member?.display_name ?? t("common.deactivated_user")}</span>
        </Link>
        <span className="flex-shrink-0 text-tertiary">{timestamp}</span>
      </div>
    </div>
  );
}
