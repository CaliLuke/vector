/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams } from "@tanstack/react-router";

import { History, MessageSquare } from "lucide-react";
// plane imports
import type { IUserActivityResponse } from "@plane/types";
import { calculateTimeAgo } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
// components
import { ActivityIcon, ActivityMessage, IssueLink } from "@/components/core/activity";
import { RichTextEditor } from "@/components/editor/rich-text";
import { ActivitySettingsLoader } from "@/components/ui/loader/settings/activity";
// hooks
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import { useCurrentUser } from "@/hooks/use-current-user";

type Props = {
  activity: IUserActivityResponse | undefined;
};

export const ActivityList = function ActivityList(props: Props) {
  const { activity } = props;
  // params
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  // derived values
  const workspaceId = workspaceDetails?.id ?? "";

  if (!activity) return <ActivitySettingsLoader />;
  return (
    <ul>
      {activity.results.map((activityItem) => (
        <ProfileActivityItem
          key={activityItem.id}
          activityItem={activityItem}
          workspaceSlug={workspaceSlug}
          workspaceId={workspaceId}
          currentUserId={currentUser?.id}
        />
      ))}
    </ul>
  );
};

type ActivityEntry = IUserActivityResponse["results"][number];
type EntryProps = { activityItem: ActivityEntry; workspaceSlug: string; workspaceId: string; currentUserId?: string };

function ProfileCommentActivity({ activityItem, workspaceSlug, workspaceId }: Omit<EntryProps, "currentUserId">) {
  return (
    <li className="mt-2">
      <div className="relative flex items-start space-x-3">
        <div className="relative px-1">
          {activityItem.new_value === "restore" && <History className="h-3.5 w-3.5 text-secondary" />}

          <span className="flex h-6 w-6 items-center justify-center rounded-full bg-layer-1 text-secondary ring-6 ring-white">
            <MessageSquare className="h-6 w-6 !text-20 text-secondary" aria-hidden="true" />
          </span>
        </div>
        <div className="min-w-0 flex-1">
          <div>
            <div className="text-11">
              {activityItem.actor_detail.is_bot
                ? activityItem.actor_detail.first_name + " Bot"
                : activityItem.actor_detail.display_name}
            </div>
            <p className="mt-0.5 text-11 text-secondary">Commented {calculateTimeAgo(activityItem.created_at)}</p>
          </div>
          <div className="issue-comments-section p-0">
            <RichTextEditor
              editable={false}
              id={activityItem.id}
              initialValue={
                (activityItem.new_value !== "" ? activityItem.new_value : activityItem.old_value)?.toString() ?? ""
              }
              containerClassName="text-11 bg-surface-1"
              workspaceId={workspaceId}
              workspaceSlug={workspaceSlug}
              projectId={activityItem.project}
            />
          </div>
        </div>
      </div>
    </li>
  );
}

function ProfileActivityMarker({ activityItem }: { activityItem: ActivityEntry }) {
  return (
    <>
      {activityItem.field ? (
        activityItem.new_value === "restore" ? (
          <History className="h-5 w-5 text-secondary" />
        ) : (
          <ActivityIcon activity={activityItem} />
        )
      ) : activityItem.actor_detail.avatar_url && activityItem.actor_detail.avatar_url !== "" ? (
        <img
          src={getFileURL(activityItem.actor_detail.avatar_url)}
          alt={activityItem.actor_detail.display_name}
          height={24}
          width={24}
          className="h-full w-full rounded-full object-cover"
        />
      ) : (
        <div className="bg-gray-700 grid h-6 w-6 place-items-center rounded-full border-2 border-white text-11 text-on-color capitalize">
          {activityItem.actor_detail.display_name?.[0]}
        </div>
      )}
    </>
  );
}

function ProfileActivityActor({ activityItem, currentUserId }: Pick<EntryProps, "activityItem" | "currentUserId">) {
  return (
    <>
      {activityItem.field === "archived_at" && activityItem.new_value !== "restore" ? (
        <span className="text-gray font-medium">Vector</span>
      ) : activityItem.actor_detail.is_bot ? (
        <span className="text-gray font-medium">{activityItem.actor_detail.first_name} Bot</span>
      ) : (
        <Link
          to="/$workspaceSlug/profile/$userId/"
          params={{
            workspaceSlug: activityItem.workspace_detail?.slug ?? "",
            userId: activityItem.actor_detail.id,
          }}
          disabled={!activityItem.workspace_detail?.slug}
          className="inline"
        >
          <span className="text-gray font-medium">
            {currentUserId === activityItem.actor_detail.id ? "You" : activityItem.actor_detail.display_name}
          </span>
        </Link>
      )}{" "}
    </>
  );
}

function ProfileActivityItem(props: EntryProps) {
  const { activityItem, currentUserId } = props;
  if (activityItem.field === "comment") return <ProfileCommentActivity {...props} />;
  if (!("field" in activityItem) || activityItem.field === "updated_by") return null;
  const message =
    activityItem.verb === "created" && !activityItem.field ? (
      <span>
        created <IssueLink activity={activityItem} />
      </span>
    ) : (
      <ActivityMessage activity={activityItem} showIssue />
    );
  return (
    <li>
      <div className="relative pb-1">
        <div className="relative flex items-start space-x-2">
          <>
            <div>
              <div className="relative mt-4 px-1.5">
                <div className="mt-1.5">
                  <div className="flex h-6 w-6 items-center justify-center">
                    <ProfileActivityMarker activityItem={activityItem} />
                  </div>
                </div>
              </div>
            </div>
            <div className="min-w-0 flex-1 border-b border-subtle py-4">
              <div className="text-13 break-words text-secondary">
                <ProfileActivityActor activityItem={activityItem} currentUserId={currentUserId} />
                <div className="inline gap-1">
                  {message}{" "}
                  <span className="flex-shrink-0 whitespace-nowrap">{calculateTimeAgo(activityItem.created_at)}</span>
                </div>
              </div>
            </div>
          </>
        </div>
      </div>
    </li>
  );
}
