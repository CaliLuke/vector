/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";

// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { EmptyStateCompact } from "@plane/propel/empty-state";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IIssueLabel } from "@plane/types";
import { Loader } from "@plane/ui";
import type { TLabelOperationsCallbacks } from "@/components/labels";
import {
  CreateUpdateLabelInline,
  DeleteLabelModal,
  ProjectSettingLabelGroup,
  ProjectSettingLabelItem,
} from "@/components/labels";
// hooks
import { useProjectLabelMutations } from "@/hooks/use-labels";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { SettingsHeading } from "../settings/heading";

export function ProjectSettingsLabelList() {
  // router
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/settings/projects/$projectId",
  });
  const { data: currentUser } = useCurrentUser();
  return (
    <ProjectLabelSettings
      key={JSON.stringify([currentUser?.id, workspaceSlug, projectId])}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
    />
  );
}

function ProjectLabelSettings({ workspaceSlug, projectId }: { workspaceSlug: string; projectId: string }) {
  const scrollToRef = useRef<HTMLDivElement>(null);
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const [showLabelForm, setLabelForm] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<IIssueLabel | null>(null);
  const [labelFormGeneration, setLabelFormGeneration] = useState(0);
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const {
    labels: projectLabels,
    updateLabelPosition,
    labelTree: projectLabelsTree,
    createLabel,
    updateLabel,
    isError: labelsError,
    isFetching: labelsFetching,
    refetch: refetchLabels,
  } = useProjectLabelMutations(workspaceSlug, projectId);
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const isEditable = canProject([EUserPermissions.ADMIN]);
  const labelOperationsCallbacks: TLabelOperationsCallbacks = {
    createLabel,
    updateLabel,
  };

  const newLabel = () => {
    setLabelFormGeneration((generation) => generation + 1);
    setIsUpdating(false);
    setLabelForm(true);
  };

  const onDrop = (
    draggingLabelId: string,
    droppedParentId: string | null,
    droppedLabelId: string | undefined,
    dropAtEndOfList: boolean
  ) => {
    if (workspaceSlug && projectId) {
      void updateLabelPosition({
        draggingLabelId,
        droppedParentId,
        droppedLabelId,
        dropAtEndOfList,
      }).catch((error) => {
        if (!mounted.current) return;
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: error?.error ?? "Label order could not be saved. Please try again.",
        });
      });
      return;
    }
  };

  return (
    <>
      {deleteDialog && (
        <DeleteLabelModal
          key={deleteDialog.id}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          data={deleteDialog}
          onClose={() => setDeleteDialog(null)}
        />
      )}
      <SettingsHeading
        title={t("project_settings.labels.heading")}
        description={t("project_settings.labels.description")}
        control={
          isEditable && (
            <Button variant="primary" size="lg" onClick={newLabel}>
              {t("common.add_label")}
            </Button>
          )
        }
      />
      <div className="mt-6 w-full">
        {showLabelForm && (
          <div className="my-2 w-full rounded-sm border border-subtle px-3.5 py-2">
            <CreateUpdateLabelInline
              key={labelFormGeneration}
              labelForm={showLabelForm}
              setLabelForm={setLabelForm}
              isUpdating={isUpdating}
              labelOperationsCallbacks={labelOperationsCallbacks}
              ref={scrollToRef}
              onClose={() => {
                setLabelForm(false);
                setIsUpdating(false);
                return true;
              }}
            />
          </div>
        )}
        {projectLabels !== undefined ? (
          projectLabels.length === 0 && !showLabelForm ? (
            <EmptyStateCompact
              assetKey="label"
              assetClassName="size-20"
              title={t("settings_empty_state.labels.title")}
              description={t("settings_empty_state.labels.description")}
              actions={[
                {
                  label: t("settings_empty_state.labels.cta_primary"),
                  onClick: () => {
                    newLabel();
                  },
                },
              ]}
              align="start"
              rootClassName="py-20"
            />
          ) : (
            projectLabelsTree?.map((label, index) => {
              if (label.children && label.children.length) {
                return (
                  <ProjectSettingLabelGroup
                    key={label.id}
                    label={label}
                    labelChildren={label.children || []}
                    handleLabelDelete={setDeleteDialog}
                    isUpdating={isUpdating}
                    setIsUpdating={setIsUpdating}
                    isLastChild={index === projectLabelsTree.length - 1}
                    onDrop={onDrop}
                    isEditable={isEditable}
                    labelOperationsCallbacks={labelOperationsCallbacks}
                  />
                );
              }
              return (
                <ProjectSettingLabelItem
                  label={label}
                  key={label.id}
                  setIsUpdating={setIsUpdating}
                  handleLabelDelete={setDeleteDialog}
                  isChild={false}
                  isLastChild={index === projectLabelsTree.length - 1}
                  onDrop={onDrop}
                  isEditable={isEditable}
                  labelOperationsCallbacks={labelOperationsCallbacks}
                />
              );
            })
          )
        ) : (
          !showLabelForm &&
          (labelsError ? (
            <div role="alert" className="flex items-center gap-3 py-8">
              <p>Unable to load project labels.</p>
              <Button variant="primary" size="sm" disabled={labelsFetching} onClick={() => void refetchLabels()}>
                Retry
              </Button>
            </div>
          ) : (
            <Loader className="space-y-5">
              <Loader.Item height="42px" />
              <Loader.Item height="42px" />
              <Loader.Item height="42px" />
              <Loader.Item height="42px" />
            </Loader>
          ))
        )}
      </div>
    </>
  );
}
