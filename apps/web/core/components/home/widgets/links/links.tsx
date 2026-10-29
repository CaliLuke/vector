/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
// computed
import { ContentOverflowWrapper } from "@/components/core/content-overflow-HOC";

import type { TLink } from "@plane/types";
import { homeLinksOptions } from "@/queries/home";
import { LinksEmptyState } from "../empty-states/links";
import { EWidgetKeys, WidgetLoader } from "../loaders";
import { ProjectLinkDetail } from "./link-detail";
import type { TLinkOperations } from "./use-links";

type TLinkOperationsModal = Exclude<TLinkOperations, "create">;

export type TProjectLinkList = {
  linkOperations: TLinkOperationsModal;
  workspaceSlug: string;
  onEdit: (link: TLink) => void;
};

export const ProjectLinkList = function ProjectLinkList(props: TProjectLinkList) {
  // props
  const { linkOperations, workspaceSlug, onEdit } = props;
  // hooks
  const query = useRecordCollection(homeLinksOptions(workspaceSlug));
  const links = query.data;
  if (query.isError)
    return (
      <div role="alert">
        Unable to load links.{" "}
        <button
          type="button"
          onClick={() => {
            void query.refetch();
          }}
        >
          Try again
        </button>
      </div>
    );

  if (links === undefined) return <WidgetLoader widgetKey={EWidgetKeys.QUICK_LINKS} />;

  if (links.length === 0) return <LinksEmptyState />;

  return (
    <div className="relative">
      <ContentOverflowWrapper
        maxHeight={150}
        containerClassName="box-border min-h-[30px] flex flex-col"
        fallback={<></>}
        buttonClassName="bg-surface-2/20"
      >
        <div className="mb-2 flex flex-1 flex-wrap gap-2">
          {links.map((link) => (
            <ProjectLinkDetail key={link.id} linkDetail={link} linkOperations={linkOperations} onEdit={onEdit} />
          ))}
        </div>
      </ContentOverflowWrapper>
    </div>
  );
};
