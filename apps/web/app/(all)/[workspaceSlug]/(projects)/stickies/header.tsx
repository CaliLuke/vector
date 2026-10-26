/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { Button } from "@plane/propel/button";
import { RecentStickyIcon } from "@plane/propel/icons";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { StickySearch } from "@/components/stickies/modal/search";
import { useStickyOperations } from "@/components/stickies/sticky/use-operations";
// hooks
import { useStickies } from "@/components/stickies/provider";

export const WorkspaceStickyHeader = function WorkspaceStickyHeader() {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug/_projects/stickies" });
  // hooks
  const { creatingSticky } = useStickies();
  const { stickyOperations } = useStickyOperations({ workspaceSlug: workspaceSlug?.toString() });

  return (
    <>
      <Header>
        <Header.LeftItem>
          <div className="flex items-center gap-2.5">
            <Breadcrumbs>
              <Breadcrumbs.Item
                component={
                  <BreadcrumbLink
                    label={`Stickies`}
                    icon={<RecentStickyIcon className="size-5 rotate-90 text-secondary" />}
                  />
                }
              />
            </Breadcrumbs>
          </div>
        </Header.LeftItem>

        <Header.RightItem>
          <StickySearch />
          <Button
            variant="primary"
            size="lg"
            onClick={() => {
              stickyOperations.create();
            }}
            loading={creatingSticky}
          >
            Add sticky
          </Button>
        </Header.RightItem>
      </Header>
    </>
  );
};
