/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { Logo } from "@plane/propel/emoji-icon-picker";
import { PageIcon } from "@plane/propel/icons";
// plane import
import type { TActivityEntityData, TPageEntityData } from "@plane/types";
import { Avatar } from "@plane/ui";
import { calculateTimeAgo } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
import { getPageName } from "@plane/utils/page";
import { ListItem } from "@/components/core/list";
// hooks
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";

type BlockProps = {
  activity: TActivityEntityData;
  parentRef: React.RefObject<HTMLDivElement | null>;
  workspaceSlug: string;
};

export function RecentPage(props: BlockProps) {
  const { activity, parentRef, workspaceSlug } = props;
  // router
  const router = useRouter();
  // store hooks
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  // derived values
  const pageDetails = activity.entity_data as TPageEntityData;

  if (!pageDetails) return <></>;

  const ownerDetails = memberById(pageDetails.owned_by)?.member;
  const pageLink = pageDetails.project_id
    ? `/${workspaceSlug}/projects/${pageDetails.project_id}/pages/${pageDetails.id}`
    : `/${workspaceSlug}/pages/${pageDetails.id}`;

  return (
    <ListItem
      key={activity.id}
      itemLink={pageLink}
      title={getPageName(pageDetails.name)}
      prependTitleElement={
        <div className="flex flex-shrink-0 items-center gap-2">
          <div className="grid size-8 flex-shrink-0 place-items-center rounded-sm bg-layer-2">
            {pageDetails?.logo_props?.in_use ? (
              <Logo logo={pageDetails?.logo_props} size={16} type="lucide" />
            ) : (
              <PageIcon className="size-4 text-tertiary" />
            )}
          </div>
          {pageDetails?.project_identifier && (
            <div className="text-13 font-medium whitespace-nowrap text-placeholder">
              {pageDetails?.project_identifier}
            </div>
          )}
        </div>
      }
      appendTitleElement={
        <div className="flex-shrink-0 text-11 font-medium text-placeholder">
          {calculateTimeAgo(activity.visited_at)}
        </div>
      }
      quickActionElement={
        <div className="flex gap-4">
          <Avatar src={getFileURL(ownerDetails?.avatar_url ?? "")} name={ownerDetails?.display_name} />
        </div>
      }
      parentRef={parentRef}
      disableLink={false}
      className="my-auto border-none !px-2 py-3"
      itemClassName="my-auto bg-layer-transparent"
      onItemClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        void router.navigate({ href: pageLink });
      }}
    />
  );
}
