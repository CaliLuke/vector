/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

import { useQuery, skipToken } from "@tanstack/react-query";
// ui
import { useTranslation } from "@plane/i18n";
import { Avatar } from "@plane/propel/avatar";
import { EmptyStateCompact } from "@plane/propel/empty-state";
import { Loader, Card } from "@plane/ui";
import { calculateTimeAgo } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
// components
import { ActivityMessage, IssueLink } from "@/components/core/activity";
// constants
import { USER_PROFILE_ACTIVITY } from "@plane/constants";
// helpers
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
// services
import { UserService } from "@/services/user.service";

const userService = new UserService();

export const ProfileActivity = function ProfileActivity() {
  const { workspaceSlug, userId } = useParams({ strict: false });
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { t } = useTranslation();

  const { data: userProfileActivity } = useQuery({
    queryKey: USER_PROFILE_ACTIVITY(workspaceSlug, userId, { per_page: 10 }),
    queryFn:
      workspaceSlug && userId
        ? () =>
            userService.getUserProfileActivity(workspaceSlug.toString(), userId.toString(), {
              per_page: 10,
            })
        : skipToken,
  });

  return (
    <div className="space-y-2">
      <h3 className="text-16 font-medium">{t("profile.stats.recent_activity.title")}</h3>
      <Card>
        {!userProfileActivity ? (
          <Loader className="space-y-5">
            <Loader.Item height="40px" />
            <Loader.Item height="40px" />
            <Loader.Item height="40px" />
            <Loader.Item height="40px" />
            <Loader.Item height="40px" />
          </Loader>
        ) : Array.isArray(userProfileActivity.results) && userProfileActivity.results.length > 0 ? (
          <div className="space-y-5">
            {userProfileActivity.results.map((activity) => (
              <div key={activity.id} className="flex gap-3">
                <Avatar
                  name={activity.actor_detail?.display_name}
                  src={getFileURL(activity.actor_detail?.avatar_url)}
                  size="base"
                  shape="square"
                />
                <div className="-mt-1 w-4/5 break-words">
                  <p className="inline text-13 text-secondary">
                    <span className="font-medium text-primary">
                      {currentUser?.id === activity.actor_detail?.id ? "You" : activity.actor_detail?.display_name}{" "}
                    </span>
                    {activity.field ? (
                      <ActivityMessage activity={activity} showIssue />
                    ) : (
                      <span>
                        created <IssueLink activity={activity} />
                      </span>
                    )}
                  </p>
                  <p className="text-11 whitespace-nowrap text-secondary">{calculateTimeAgo(activity.created_at)}</p>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <EmptyStateCompact title={t("no_data_yet")} assetKey="unknown" assetClassName="size-20" />
        )}
      </Card>
    </div>
  );
};
