/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { ModuleStatusSelect } from "@/components/modules/select/status";

import { useMutation } from "@tanstack/react-query";
import { Info, SquareUser } from "lucide-react";
import { Disclosure, Transition } from "@headlessui/react";
import { EUserPermissions, EEstimateSystem } from "@plane/constants";
// plane types
import { useTranslation } from "@plane/i18n";
import {
  PlusIcon,
  MembersPropertyIcon,
  WorkItemsIcon,
  StartDatePropertyIcon,
  ChevronDownIcon,
  ChevronRightIcon,
} from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ILinkDetails, IModule, ModuleLink } from "@plane/types";
// plane ui
import { Loader, TextArea } from "@plane/ui";
// components
// helpers
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { CreateUpdateModuleLinkModal, ModuleAnalyticsProgress, ModuleLinksList } from "@/components/modules";
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";
import { useModuleDetails, useProjectModules } from "@/hooks/use-modules";
import { moduleLinkMutationOptions } from "@/queries/module-links";
import { useQueryClient } from "@tanstack/react-query";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { captureParentAccess } from "@/queries/cycle-module-scope";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useProjectPermissions } from "@/hooks/use-permissions";
type Props = {
  moduleId: string;
  workspaceSlug: string;
  projectId: string;
  handleClose: () => void;
  isArchived?: boolean;
};

function ModuleCountRow(props: { label: string; count: string }) {
  const { label, count } = props;
  return (
    <div className="flex items-center justify-start gap-1">
      <div className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
        <WorkItemsIcon className="h-4 w-4" />
        <span className="text-14">{label}</span>
      </div>
      <div className="flex h-7 w-3/5 items-center">
        <span className="px-1.5 text-13 text-tertiary">{count}</span>
      </div>
    </div>
  );
}

function ModuleEstimatePointsRow(props: { visible: boolean; count: string; translate: (key: string) => string }) {
  const { visible, count, translate } = props;
  if (!visible) return null;
  return <ModuleCountRow label={translate("points")} count={count} />;
}

export const ModuleAnalyticsSidebar = function ModuleAnalyticsSidebar(props: Props) {
  const { moduleId, handleClose, isArchived, workspaceSlug, projectId } = props;
  // states
  const [moduleLinkModal, setModuleLinkModal] = useState(false);
  const [selectedLinkToUpdate, setSelectedLinkToUpdate] = useState<ILinkDetails | null>(null);
  // router
  const detail = useModuleDetails(workspaceSlug, projectId, moduleId, isArchived),
    catalogue = useProjectModules(workspaceSlug, projectId, Boolean(isArchived)),
    client = useQueryClient();
  const current = useCommittedScope(JSON.stringify([detail.scope, moduleId]));
  const access = captureParentAccess(client, detail.scope);
  const links = useMutation(moduleLinkMutationOptions(client, detail.scope, moduleId, detail.queryKey));

  // store hooks
  const { t } = useTranslation();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const { isEnabled: areEstimateEnabled, activeEstimate: estimateDetails } = useProjectEstimates(
    workspaceSlug,
    projectId
  );

  // derived values
  const moduleDetails = detail.module;
  const estimateType = areEstimateEnabled ? estimateDetails : undefined;
  const isEstimatePointValid = estimateType?.type === EEstimateSystem.POINTS;

  const update = catalogue.mutation;
  const updateModule = (data: Partial<IModule>) =>
    update.mutate(
      { type: "patch", id: moduleId, data },
      {
        onSuccess: (result) => {
          if (current() && result.captured.isCurrent())
            setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Module updated successfully." });
        },
        onError: () => {
          if (current() && access.isCurrent())
            setToast({ type: TOAST_TYPE.ERROR, title: "Unable to save changes", message: "Please try again." });
        },
      }
    );
  const handleCreateLink = (data: ModuleLink) => links.mutateAsync({ type: "create", data });
  const handleUpdateLink = (data: ModuleLink, id: string) => links.mutateAsync({ type: "patch", id, data });
  const handleDeleteLink = async (id: string) => {
    try {
      const result = await links.mutateAsync({ type: "delete", id });
      if (current() && result.captured.isCurrent())
        setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Module link deleted successfully." });
    } catch (error) {
      if (current() && access.isCurrent())
        setToast({ type: TOAST_TYPE.ERROR, title: "Unable to delete link", message: "Please try again." });
      console.warn("The module link could not be deleted.", error);
    }
  };

  const handleEditLink = (link: ILinkDetails) => {
    setSelectedLinkToUpdate(link);
    setModuleLinkModal(true);
  };

  if (detail.isError || detail.isAccessDenied) return <CatalogueReadFailure label="modules" retry={detail.retry} />;
  if (!moduleDetails)
    return (
      <Loader>
        <div className="space-y-2">
          <Loader.Item height="15px" width="50%" />
          <Loader.Item height="15px" width="30%" />
        </div>
        <div className="mt-8 space-y-3">
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
        </div>
      </Loader>
    );

  const issueCount =
    moduleDetails.total_issues === 0
      ? "0 work items"
      : `${moduleDetails.completed_issues}/${moduleDetails.total_issues}`;

  const issueEstimatePointCount =
    moduleDetails.total_estimate_points === 0
      ? "0 work items"
      : `${moduleDetails.completed_estimate_points}/${moduleDetails.total_estimate_points}`;

  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const controlsDisabled = !isEditingAllowed || Boolean(isArchived) || update.isPending;

  return (
    <div className="relative">
      {moduleLinkModal && (
        <CreateUpdateModuleLinkModal
          key={JSON.stringify([detail.scope, moduleId, selectedLinkToUpdate?.id])}
          isOpen={moduleLinkModal}
          handleClose={() => {
            setModuleLinkModal(false);
            setSelectedLinkToUpdate(null);
          }}
          data={selectedLinkToUpdate}
          createLink={handleCreateLink}
          updateLink={handleUpdateLink}
        />
      )}
      <>
        <div className={`sticky top-0 z-10 flex items-center justify-between bg-surface-1 pt-5 pb-5`}>
          <div>
            <button
              className="flex h-5 w-5 items-center justify-center rounded-full bg-layer-3"
              onClick={() => handleClose()}
            >
              <ChevronRightIcon className="h-3 w-3 stroke-2 text-on-color" />
            </button>
          </div>
        </div>

        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-5 pt-2">
            <ModuleStatusSelect
              badge
              value={moduleDetails.status}
              disabled={controlsDisabled}
              onChange={(status) => updateModule({ status })}
            />
          </div>
          <h4 className="w-full text-18 font-semibold break-words text-primary">{moduleDetails.name}</h4>
        </div>

        {moduleDetails.description && (
          <TextArea
            className="ring-none !m-0 max-h-max w-full resize-none !border-0 bg-transparent !p-0 text-13 leading-5 text-secondary outline-none"
            value={moduleDetails.description}
            disabled
          />
        )}

        <div className="flex flex-col gap-5 pt-2.5 pb-6">
          <ModulePropertyControls
            workspaceSlug={workspaceSlug}
            module={moduleDetails}
            disabled={controlsDisabled}
            onChange={updateModule}
          />
          <ModuleCountRow label={t("issues")} count={issueCount} />

          {/**
           * NOTE: Render this section when estimate points of he projects is enabled and the estimate system is points
           */}
          <ModuleEstimatePointsRow visible={isEstimatePointValid} count={issueEstimatePointCount} translate={t} />
        </div>

        <ModuleAnalyticsProgress
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          moduleId={moduleDetails.id}
          isArchived={isArchived}
        />

        <div className="flex flex-col">
          <div className="flex w-full flex-col items-center justify-start gap-2 border-t border-subtle px-1.5 py-5">
            <Disclosure defaultOpen={!!moduleDetails.link_module?.length}>
              {({ open }) => (
                <div className={`relative flex h-full w-full flex-col ${open ? "" : "flex-row"}`}>
                  <Disclosure.Button className="flex w-full items-center justify-between gap-2 p-1.5">
                    <div className="flex items-center justify-start gap-2 text-13">
                      <span className="font-medium text-secondary">{t("common.links")}</span>
                    </div>
                    <div className="flex items-center gap-2.5">
                      <ChevronDownIcon
                        className={`h-3.5 w-3.5 ${open ? "rotate-180 transform" : ""}`}
                        aria-hidden="true"
                      />
                    </div>
                  </Disclosure.Button>
                  <Transition show={open}>
                    <Disclosure.Panel>
                      <div className="mt-2 flex min-h-72 w-full flex-col space-y-3 overflow-y-auto">
                        {isEditingAllowed && moduleDetails.link_module && moduleDetails.link_module.length > 0 ? (
                          <>
                            {isEditingAllowed && !isArchived && (
                              <div className="flex w-full items-center justify-end">
                                <button
                                  className="flex items-center gap-1.5 text-13 font-medium text-accent-primary"
                                  onClick={() => setModuleLinkModal(true)}
                                >
                                  <PlusIcon className="h-3 w-3" />
                                  {t("add_link")}
                                </button>
                              </div>
                            )}

                            {moduleId && (
                              <ModuleLinksList
                                links={moduleDetails.link_module ?? []}
                                workspaceSlug={workspaceSlug}
                                handleEditLink={handleEditLink}
                                handleDeleteLink={handleDeleteLink}
                                disabled={controlsDisabled}
                              />
                            )}
                          </>
                        ) : (
                          <div className="flex items-center justify-between gap-2">
                            <div className="flex items-center gap-2">
                              <Info className="h-3.5 w-3.5 stroke-[1.5] text-tertiary" />
                              <span className="p-0.5 text-11 text-tertiary">{t("common.no_links_added_yet")}</span>
                            </div>
                            {isEditingAllowed && !isArchived && (
                              <button
                                className="flex items-center gap-1.5 text-13 font-medium text-accent-primary"
                                onClick={() => setModuleLinkModal(true)}
                              >
                                <PlusIcon className="h-3 w-3" />
                                {t("add_link")}
                              </button>
                            )}
                          </div>
                        )}
                      </div>
                    </Disclosure.Panel>
                  </Transition>
                </div>
              )}
            </Disclosure>
          </div>
        </div>
      </>
    </div>
  );
};

const ModulePropertyControls = function ModulePropertyControls({
  workspaceSlug,
  module,
  disabled,
  onChange,
}: {
  workspaceSlug: string | undefined;
  module: IModule;
  disabled: boolean;
  onChange: (patch: Partial<IModule>) => void;
}) {
  const { t } = useTranslation();
  return (
    <>
      <div className="flex items-center justify-start gap-1">
        <div className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
          <StartDatePropertyIcon className="h-4 w-4" />
          <span className="text-14">{t("date_range")}</span>
        </div>
        <div className="h-7">
          <DateRangeDropdown
            buttonContainerClassName="w-full"
            buttonVariant="background-with-text"
            value={{ from: getDate(module.start_date), to: getDate(module.target_date) }}
            onSelect={(value) =>
              onChange({
                start_date: value?.from ? renderFormattedPayloadDate(value.from) : null,
                target_date: value?.to ? renderFormattedPayloadDate(value.to) : null,
              })
            }
            placeholder={{ from: t("start_date"), to: t("end_date") }}
            disabled={disabled}
          />
        </div>
      </div>
      <div className="flex items-center justify-start gap-1">
        <div className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
          <SquareUser className="h-4 w-4" />
          <span className="text-14">{t("lead")}</span>
        </div>

        <div className="h-7 w-3/5">
          <MemberDropdown
            workspaceSlug={workspaceSlug}
            value={module.lead_id ?? null}
            onChange={(val) => {
              onChange({ lead_id: val });
            }}
            projectId={module.project_id}
            multiple={false}
            buttonVariant="background-with-text"
            placeholder={t("lead")}
            disabled={disabled}
            icon={SquareUser}
          />
        </div>
      </div>
      <div className="flex items-center justify-start gap-1">
        <div className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
          <MembersPropertyIcon className="h-4 w-4" />
          <span className="text-14">{t("members")}</span>
        </div>

        <div className="h-7 w-3/5">
          <MemberDropdown
            workspaceSlug={workspaceSlug}
            value={module.member_ids ?? []}
            onChange={(val: string[]) => {
              onChange({ member_ids: val });
            }}
            multiple
            projectId={module.project_id}
            buttonVariant={module.member_ids?.length > 0 ? "transparent-without-text" : "background-with-text"}
            buttonClassName={module.member_ids?.length > 0 ? "hover:bg-transparent px-0" : ""}
            disabled={disabled}
          />
        </div>
      </div>
    </>
  );
};
