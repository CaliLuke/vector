/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useEffect } from "react";

import { useQuery, skipToken } from "@tanstack/react-query";
// services
import { USER_PROFILE_ACTIVITY } from "@plane/constants";
import { UserService } from "@/services/user.service";
// components
import { ActivityList } from "./activity-list";
// query keys

// services
const userService = new UserService();

type Props = {
  cursor: string;
  perPage: number;
  updateResultsCount: (count: number) => void;
  updateTotalPages: (count: number) => void;
};

export function WorkspaceActivityListPage(props: Props) {
  const { cursor, perPage, updateResultsCount, updateTotalPages } = props;
  // router
  const { workspaceSlug, userId } = useParams({ strict: false });

  const { data: userProfileActivity } = useQuery({
    queryKey: USER_PROFILE_ACTIVITY(workspaceSlug, userId, {
      per_page: perPage,
      cursor,
    }),
    queryFn:
      workspaceSlug && userId
        ? () =>
            userService.getUserProfileActivity(workspaceSlug.toString(), userId.toString(), {
              cursor,
              per_page: perPage,
            })
        : skipToken,
  });

  useEffect(() => {
    if (!userProfileActivity) return;

    updateTotalPages(userProfileActivity.total_pages);
    updateResultsCount(userProfileActivity.results.length);
  }, [updateResultsCount, updateTotalPages, userProfileActivity]);

  return <ActivityList activity={userProfileActivity} />;
}
