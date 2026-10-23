/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import React from "react";
// components
import { PageHead } from "@/components/core/page-title";
import { ProfileIssuesPage } from "@/components/profile/profile-issues";

const ProfilePageHeader = {
  assigned: "Profile - Assigned",
  created: "Profile - Created",
  subscribed: "Profile - Subscribed",
};

function isValidProfileViewId(viewId: string): viewId is keyof typeof ProfilePageHeader {
  return viewId in ProfilePageHeader;
}

function ProfileIssuesTypePage(): React.JSX.Element | null {
  const { profileViewId } = Route.useParams();

  if (!isValidProfileViewId(profileViewId)) return null;

  const header = ProfilePageHeader[profileViewId];

  return (
    <>
      <PageHead title={header} />
      <ProfileIssuesPage type={profileViewId} />
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/profile/$userId/$profileViewId")({
  component: ProfileIssuesTypePage,
});
