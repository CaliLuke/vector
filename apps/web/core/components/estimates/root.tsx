/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
// components
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
import { SettingsHeading } from "@/components/settings/heading";
// hooks
import { EmptyStateCompact } from "@plane/propel/empty-state";
import { useEstimateEditorKey, useProjectEstimates } from "@/hooks/use-estimates";
// local imports
import { CreateEstimateModal } from "./create/modal";
import { DeleteEstimateModal } from "./delete/modal";
import { EstimateDisableSwitch } from "./estimate-disable-switch";
import { EstimateList } from "./estimate-list";
import { EstimateLoaderScreen } from "./loader-screen";

type TEstimateRoot = {
  workspaceSlug: string;
  projectId: string;
  isAdmin: boolean;
};

export function EstimateRoot(props: TEstimateRoot) {
  const key = useEstimateEditorKey(props.workspaceSlug, props.projectId);
  return <EstimateSettings key={key} {...props} />;
}

const EstimateSettings = function EstimateSettings(props: TEstimateRoot) {
  const { workspaceSlug, projectId, isAdmin } = props;
  // hooks
  const estimates = useProjectEstimates(workspaceSlug, projectId);
  const currentActiveEstimateId = estimates.activeEstimate?.id;
  const archivedEstimates = estimates.archivedEstimates;
  // states
  const [isEstimateCreateModalOpen, setIsEstimateCreateModalOpen] = useState(false);
  const [estimateToDelete, setEstimateToDelete] = useState<string | undefined>();

  const { t } = useTranslation();

  if (estimates.isPending) return <EstimateLoaderScreen />;
  if (estimates.isError || estimates.isAccessDenied)
    return (
      <div role="alert">
        Estimates are unavailable.{" "}
        <button
          type="button"
          onClick={() => {
            void estimates.retry();
          }}
        >
          Retry
        </button>
      </div>
    );

  return (
    <>
      <div>
        {/* header */}
        <SettingsHeading
          title={t("project_settings.estimates.heading")}
          description={t("project_settings.estimates.description")}
        />
        <div className="mt-6">
          {/* current active estimate section */}
          {currentActiveEstimateId ? (
            <>
              {/* estimates activated deactivated section */}
              <SettingsBoxedControlItem
                title={t("project_settings.estimates.title")}
                description={t("project_settings.estimates.enable_description")}
                control={
                  <EstimateDisableSwitch workspaceSlug={workspaceSlug} projectId={projectId} isAdmin={isAdmin} />
                }
              />
              {/* active estimates section */}
              <div className="mt-12 flex flex-col gap-y-4">
                <SettingsHeading title="Estimates list" variant="h6" />
                <EstimateList
                  estimates={estimates.activeEstimate ? [estimates.activeEstimate] : []}
                  isAdmin={isAdmin}
                  isEditable
                  onDeleteClick={(estimateId: string) => setEstimateToDelete(estimateId)}
                />
              </div>
            </>
          ) : (
            <EmptyStateCompact
              assetKey="estimate"
              assetClassName="size-20"
              title={t("settings_empty_state.estimates.title")}
              description={t("settings_empty_state.estimates.description")}
              actions={[
                {
                  label: t("settings_empty_state.estimates.cta_primary"),
                  onClick: () => setIsEstimateCreateModalOpen(true),
                },
              ]}
              align="start"
              rootClassName="py-20"
            />
          )}
          {/* archived estimates section */}
          {archivedEstimates && archivedEstimates.length > 0 && (
            <div className="mt-12 flex flex-col gap-y-4">
              <SettingsHeading
                title="Archived estimates"
                description={
                  <>
                    Estimates have gone through a change, these are the estimates you had in your older versions which
                    were not in use. Read more about them&nbsp;
                    <a
                      href={"https://github.com/CaliLuke/vector/blob/main/docs/README.md"}
                      target="_blank"
                      className="text-accent-primary/80 hover:text-accent-primary"
                      rel="noreferrer"
                    >
                      here.
                    </a>
                  </>
                }
                variant="h6"
              />
              <EstimateList estimates={archivedEstimates} isAdmin={isAdmin} />
            </div>
          )}
        </div>
      </div>
      {/* CRUD modals */}
      <CreateEstimateModal
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isOpen={isEstimateCreateModalOpen}
        handleClose={() => setIsEstimateCreateModalOpen(false)}
      />
      <DeleteEstimateModal
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        estimateId={estimateToDelete}
        isOpen={Boolean(estimateToDelete)}
        handleClose={() => setEstimateToDelete(undefined)}
      />
    </>
  );
};
