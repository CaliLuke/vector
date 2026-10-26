/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";

import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { ToggleSwitch } from "@plane/ui";
// hooks
import { useEstimateMutations } from "@/hooks/use-estimates";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
// i18n
type TEstimateDisableSwitch = {
  workspaceSlug: string;
  projectId: string;
  isAdmin: boolean;
};

export const EstimateDisableSwitch = function EstimateDisableSwitch(props: TEstimateDisableSwitch) {
  const { workspaceSlug, projectId, isAdmin } = props;
  // i18n
  const { t } = useTranslation();
  // hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const currentProjectDetails = projectRows?.find((project) => project.id === projectId);
  const { activeEstimate, mutation, isPending, isError, isAccessDenied } = useEstimateMutations(
    workspaceSlug,
    projectId
  );
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const currentProjectActiveEstimate = currentProjectDetails?.estimate || undefined;

  const disableEstimate = async () => {
    if (!workspaceSlug || !projectId) return;

    try {
      const result = await mutation.mutateAsync({
        type: "select",
        expectedId: currentProjectActiveEstimate ?? null,
        estimateId: currentProjectActiveEstimate ? null : (activeEstimate?.id ?? null),
      });
      if (!mounted.current || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: currentProjectActiveEstimate
          ? t("project_settings.estimates.toasts.disabled.success.title")
          : t("project_settings.estimates.toasts.enabled.success.title"),
        message: currentProjectActiveEstimate
          ? t("project_settings.estimates.toasts.disabled.success.message")
          : t("project_settings.estimates.toasts.enabled.success.message"),
      });
    } catch (_err) {
      if (!mounted.current) return;
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("project_settings.estimates.toasts.disabled.error.title"),
        message: t("project_settings.estimates.toasts.disabled.error.message"),
      });
    }
  };

  return (
    <ToggleSwitch
      value={Boolean(currentProjectActiveEstimate)}
      onChange={disableEstimate}
      disabled={!isAdmin || mutation.isPending || isPending || isError || isAccessDenied || !activeEstimate}
      size="sm"
    />
  );
};
