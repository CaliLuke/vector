/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

import { SquareUser } from "lucide-react";
// Vector imports
import { MODULE_STATUS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Tooltip } from "@plane/propel/tooltip";
import type { IModule } from "@plane/types";
// components
import { ModuleQuickActions } from "@/components/modules";
import { ModuleDateRangeField } from "@/components/modules/module-date-range-field";
import { ModuleStatusDropdown } from "@/components/modules/module-status-dropdown";
import { useModuleActions } from "@/components/modules/use-module-actions";
// hooks
import { AuthorizedFavoriteStar } from "@/components/common/authorized-favorite-star";
import { ButtonAvatars } from "../dropdowns/member/avatar";

type Props = {
  moduleId: string;
  moduleDetails: IModule;
  parentRef: React.RefObject<HTMLDivElement | null>;
};

export const ModuleListItemAction = function ModuleListItemAction(props: Props) {
  const { moduleId, moduleDetails, parentRef } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { projectId } = useParams({ strict: false });
  //   store hooks
  const {
    handleAddToFavorites,
    handleRemoveFromFavorites,
    handleModuleDetailsChange,
    isFavorite,
    isFavoritesReady,
    isEditingAllowed,
    getUserDetails,
  } = useModuleActions({
    moduleId,
    workspaceSlug,
    projectId,
  });

  const { t } = useTranslation();

  // derived values

  const moduleStatus = MODULE_STATUS.find((status) => status.value === moduleDetails.status);
  const isDisabled = !isEditingAllowed || !!moduleDetails?.archived_at;

  // handlers
  const moduleLeadDetails = moduleDetails.lead_id ? getUserDetails(moduleDetails.lead_id) : undefined;

  return (
    <>
      <ModuleDateRangeField
        moduleDetails={moduleDetails}
        isDisabled={isDisabled}
        handleModuleDetailsChange={handleModuleDetailsChange}
        mergeDates
        placeholder={{ from: t("start_date"), to: t("end_date") }}
      />

      {moduleStatus && (
        <ModuleStatusDropdown
          isDisabled={isDisabled}
          moduleDetails={moduleDetails}
          handleModuleDetailsChange={handleModuleDetailsChange}
        />
      )}

      {moduleLeadDetails ? (
        <span className="cursor-default">
          <ButtonAvatars showTooltip={false} userIds={moduleLeadDetails?.id} workspaceSlug={workspaceSlug} />
        </span>
      ) : (
        <Tooltip tooltipContent="No lead">
          <SquareUser className="h-4 w-4 text-tertiary" />
        </Tooltip>
      )}

      <AuthorizedFavoriteStar
        canFavorite={isEditingAllowed}
        isReady={isFavoritesReady}
        disabled={!!moduleDetails.archived_at}
        onAdd={handleAddToFavorites}
        onRemove={handleRemoveFromFavorites}
        selected={isFavorite}
      />
      {workspaceSlug && projectId && (
        <div className="hidden md:block">
          <ModuleQuickActions
            module={moduleDetails}
            parentRef={parentRef}
            moduleId={moduleId}
            projectId={projectId.toString()}
            workspaceSlug={workspaceSlug.toString()}
          />
        </div>
      )}
    </>
  );
};
