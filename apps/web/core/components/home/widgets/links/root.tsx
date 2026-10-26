/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";

import { useTranslation } from "@plane/i18n";
import { PlusIcon } from "@plane/propel/icons";
import type { THomeWidgetProps, TLink } from "@plane/types";
import { LinkCreateUpdateModal } from "./create-update-link-modal";
import { ProjectLinkList } from "./links";
import { useLinks } from "./use-links";

export const DashboardQuickLinks = function DashboardQuickLinks(props: THomeWidgetProps) {
  const { workspaceSlug } = props;
  const [isLinkModalOpen, toggleLinkModal] = useState(false);
  const [linkData, setLinkData] = useState<TLink>();
  const closeModal = useCallback(() => toggleLinkModal(false), []);
  const { linkOperations } = useLinks(workspaceSlug, closeModal);
  const { t } = useTranslation();

  const handleCreateLinkModal = useCallback(() => {
    toggleLinkModal(true);
    setLinkData(undefined);
  }, [toggleLinkModal, setLinkData]);

  return (
    <>
      <LinkCreateUpdateModal
        isModalOpen={isLinkModalOpen}
        handleOnClose={() => toggleLinkModal(false)}
        linkOperations={linkOperations}
        preloadedData={linkData}
      />
      <div className="mb-2">
        <div className="mb-4 flex items-center justify-between">
          <div className="text-14 font-semibold text-tertiary">{t("home.quick_links.title_plural")}</div>
          <button
            onClick={handleCreateLinkModal}
            className="my-auto flex gap-1 text-13 font-medium text-accent-primary"
          >
            <PlusIcon className="my-auto size-4" /> <span>{t("home.quick_links.add")}</span>
          </button>
        </div>
        <div className="flex w-full flex-wrap">
          {/* rendering links */}
          <ProjectLinkList
            workspaceSlug={workspaceSlug}
            linkOperations={linkOperations}
            onEdit={(link) => {
              setLinkData(link);
              toggleLinkModal(true);
            }}
          />
        </div>
      </div>
    </>
  );
};
