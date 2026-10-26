/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { Logo } from "@plane/propel/emoji-icon-picker";
import { PageIcon } from "@plane/propel/icons";
// plane imports
import { getPageName } from "@plane/utils/page";
import type { TPage } from "@plane/types";
// components
import { ListItem } from "@/components/core/list";
import { BlockItemAction } from "@/components/pages/list/block-item-action";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// plane web hooks
import { pageRedirectLink } from "@/helpers/page-permissions";

type TPageListBlock = {
  page: TPage;
  projectId: string;
  workspaceSlug: string;
};

export function PageListBlock(props: TPageListBlock) {
  const { page, projectId, workspaceSlug } = props;
  // refs
  const parentRef = useRef(null);
  // hooks
  const { isMobile } = usePlatformOS();
  // handle page check
  // derived values
  const { name, logo_props } = page;

  return (
    <ListItem
      prependTitleElement={
        <>
          {logo_props?.in_use ? (
            <Logo logo={logo_props} size={16} type="lucide" />
          ) : (
            <PageIcon className="h-4 w-4 text-tertiary" />
          )}
        </>
      }
      title={getPageName(name)}
      itemLink={pageRedirectLink(workspaceSlug, projectId, page.id)}
      actionableItems={
        <BlockItemAction page={page} parentRef={parentRef} projectId={projectId} workspaceSlug={workspaceSlug} />
      }
      isMobile={isMobile}
      parentRef={parentRef}
    />
  );
}
