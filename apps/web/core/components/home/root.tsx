/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { ContentWrapper } from "@plane/ui";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { useCurrentUserProfile, useUpdateUserTourCompleted } from "@/hooks/use-current-user-profile";
// plane web imports
import { TourRoot } from "@/components/onboarding/tour/root";
// local imports
import { DashboardWidgets } from "./home-dashboard-widgets";
import { UserGreetingsView } from "../user/user-greetings";
import { HomePeekOverviewsRoot } from "../issues/peek-overview/peek-overviews";

export const WorkspaceHomeView = function WorkspaceHomeView() {
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { data: currentUserProfile } = useCurrentUserProfile();
  const { mutateAsync: updateTourCompleted } = useUpdateUserTourCompleted();

  const handleTourCompleted = async () => {
    try {
      await updateTourCompleted();
    } catch (error) {
      console.error("Error updating tour completed", error);
    }
  };

  return (
    <>
      {currentUserProfile && !currentUserProfile.is_tour_completed && (
        <div className="fixed top-0 left-0 z-20 grid h-full w-full place-items-center overflow-y-auto bg-backdrop transition-opacity">
          <TourRoot onComplete={handleTourCompleted} />
        </div>
      )}
      <>
        <HomePeekOverviewsRoot />
        <ContentWrapper className="mx-auto scrollbar-hide gap-6 bg-surface-1 px-page-x">
          <div className="mx-auto w-full max-w-[800px]">
            {currentUser && <UserGreetingsView user={currentUser} />}
            <DashboardWidgets />
          </div>
        </ContentWrapper>
      </>
    </>
  );
};
