/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute } from "@tanstack/react-router";
import { PageHead } from "@/components/core/page-title";
import { StickiesInfinite } from "@/components/stickies/layout/stickies-infinite";

function WorkspaceStickiesPage(): React.JSX.Element | null {
  return (
    <>
      <PageHead title="Your stickies" />
      <div className="relative h-full w-full overflow-hidden overflow-y-auto">
        <StickiesInfinite />
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/stickies/")({
  component: WorkspaceStickiesPage,
});
