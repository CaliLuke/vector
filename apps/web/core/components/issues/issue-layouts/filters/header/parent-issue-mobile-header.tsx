/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";
import { useTranslation } from "@plane/i18n";
import { ChevronDownIcon } from "@plane/propel/icons";
import type { ICycle, EIssueLayoutTypes, IModule, IProject } from "@plane/types";
import { WorkItemsModal } from "@/components/analytics/work-items/modal";
import { ParentIssueDisplaySelection } from "./display-filters/parent-issue-display-selection";
import { FiltersDropdown } from "./helpers/dropdown";
import { MobileLayoutSelection } from "./mobile-layout-selection";
import { PARENT_ISSUE_MOBILE_LAYOUTS, persistParentIssueLayout } from "./parent-issue-layout-selection";

type Props = {
  workspaceSlug: string;
  projectDetails: IProject | undefined;
  cycleDetails?: ICycle;
  moduleDetails?: IModule;
  canUpdateLayout: boolean;
  setDisplay: (patch: { layout: EIssueLayoutTypes }) => Promise<unknown>;
  showLayoutLabel?: boolean;
};

export function ParentIssueMobileHeader({
  workspaceSlug,
  projectDetails,
  cycleDetails,
  moduleDetails,
  canUpdateLayout,
  setDisplay,
  showLayoutLabel = false,
}: Props) {
  const [analyticsModal, setAnalyticsModal] = useState(false);
  const { t } = useTranslation();
  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      if (!canUpdateLayout) return;
      void persistParentIssueLayout(setDisplay, layout);
    },
    [canUpdateLayout, setDisplay]
  );

  return (
    <div className="block md:hidden">
      <WorkItemsModal
        workspaceSlug={workspaceSlug}
        projectDetails={projectDetails}
        cycleDetails={cycleDetails}
        moduleDetails={moduleDetails}
        isOpen={analyticsModal}
        onClose={() => setAnalyticsModal(false)}
      />
      <div className="flex justify-evenly border-b border-subtle bg-surface-1 py-2">
        <MobileLayoutSelection
          layouts={PARENT_ISSUE_MOBILE_LAYOUTS}
          onChange={handleLayoutChange}
          button={
            showLayoutLabel ? (
              <span className="flex flex-grow justify-center text-13 text-secondary">{t("common.layout")}</span>
            ) : undefined
          }
        />
        <div className="flex flex-grow items-center justify-center border-l border-subtle text-13 text-secondary">
          <FiltersDropdown
            title={t("common.display")}
            placement="bottom-end"
            menuButton={
              <span className="flex items-center text-13 text-secondary">
                {t("common.display")}
                <ChevronDownIcon className="ml-2 h-4 w-4 text-secondary" />
              </span>
            }
          >
            <ParentIssueDisplaySelection project={projectDetails} />
          </FiltersDropdown>
        </div>
        <button
          type="button"
          onClick={() => setAnalyticsModal(true)}
          className="flex flex-grow justify-center border-l border-subtle text-13 text-secondary"
        >
          {t("common.analytics")}
        </button>
      </div>
    </div>
  );
}
