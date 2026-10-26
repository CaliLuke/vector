/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { hashKey, useMutation } from "@tanstack/react-query";
import { Rocket } from "lucide-react";
import { Combobox } from "@headlessui/react";
// i18n
import { useTranslation } from "@plane/i18n";
// types
import { Button } from "@plane/propel/button";
import { SearchIcon, CloseIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { ISearchIssueResponse, TProjectIssuesSearchParams } from "@plane/types";
// ui
import { Loader, ToggleSwitch, EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { generateWorkItemLink } from "@plane/utils/work-item";
import { getTabIndex } from "@plane/utils/tab-indices";
// helpers
// hooks
import useDebounce from "@/hooks/use-debounce";
import { usePlatformOS } from "@/hooks/use-platform-os";
// services
import { useProjectIssueSearch } from "@/hooks/use-project-issue-search";
// components
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
import { IssueSearchModalEmptyState } from "./issue-search-modal-empty-state";

type Props = {
  workspaceSlug: string | undefined;
  projectId?: string;
  isOpen: boolean;
  handleClose: () => void;
  searchParams: Partial<TProjectIssuesSearchParams>;
  handleOnSubmit: (data: ISearchIssueResponse[]) => Promise<void>;
  workspaceLevelToggle?: boolean;
  shouldHideIssue?: (issue: ISearchIssueResponse) => boolean;
};

export function ExistingIssuesListModal(props: Props) {
  return props.isOpen ? (
    <IssueSelectionDialog key={hashKey([props.workspaceSlug, props.projectId, props.searchParams])} {...props} />
  ) : null;
}

function visibleSearchIssues(issues: ISearchIssueResponse[] | undefined, shouldHideIssue: Props["shouldHideIssue"]) {
  return (issues ?? []).filter((issue) => !shouldHideIssue?.(issue));
}

function IssueSelectionDialog({
  workspaceSlug,
  projectId,
  handleClose: onClose,
  searchParams,
  handleOnSubmit,
  workspaceLevelToggle = false,
  shouldHideIssue,
}: Props) {
  const { t } = useTranslation();
  const [searchTerm, setSearchTerm] = useState(searchParams.search ?? "");
  const [isWorkspaceLevel, setIsWorkspaceLevel] = useState(false);
  const { isMobile } = usePlatformOS();
  const debouncedSearchTerm = useDebounce(searchTerm, 500);
  const { baseTabIndex } = getTabIndex(undefined, isMobile);
  const search = useProjectIssueSearch(workspaceSlug, projectId, {
    ...searchParams,
    search: debouncedSearchTerm,
    workspace_search: isWorkspaceLevel,
  });
  const submission = useMutation({ mutationFn: handleOnSubmit, retry: false });
  const form = useForm({
    defaultValues: { selectedIssues: [] as ISearchIssueResponse[] },
    validators: { onSubmit: ({ value }) => (value.selectedIssues.length ? undefined : t("issue.select.error")) },
    onSubmit: async ({ value }) => {
      if (!search.accessReady) throw new Error("Work item access changed. Retry the search.");
      await submission.mutateAsync(value.selectedIssues, { onSuccess: () => onClose() });
    },
  });
  const { selectedIssues, isSubmitting, errors } = useStore(form.store, (state) => ({
    selectedIssues: state.values.selectedIssues,
    isSubmitting: state.isSubmitting,
    errors: state.errors,
  }));
  const filteredIssues = visibleSearchIssues(search.data, shouldHideIssue);
  const isSearching = search.isFetching || searchTerm !== debouncedSearchTerm;
  const allVisibleSelected =
    filteredIssues.length > 0 &&
    filteredIssues.every((issue) => selectedIssues.some((selected) => selected.id === issue.id));
  const handleClose = () => {
    if (!form.state.isSubmitting) onClose();
  };
  const onSubmit = () => {
    if (form.state.isSubmitting) return;
    void form
      .handleSubmit()
      .catch(() => form.setErrorMap({ onSubmit: "Could not add the selected work items. Please try again." }));
  };
  const handleSelectIssues = () => {
    form.setFieldValue("selectedIssues", (selected) =>
      allVisibleSelected
        ? selected.filter((issue) => !filteredIssues.some((visible) => visible.id === issue.id))
        : [...selected, ...filteredIssues.filter((issue) => !selected.some((item) => item.id === issue.id))]
    );
  };

  return (
    <ModalCore isOpen handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <Combobox
        as="div"
        multiple
        by="id"
        value={selectedIssues}
        disabled={isSubmitting}
        onChange={(issues: ISearchIssueResponse[]) => form.setFieldValue("selectedIssues", issues)}
      >
        <div className="relative m-1">
          <SearchIcon
            className="text-opacity-40 pointer-events-none absolute top-3.5 left-4 h-5 w-5 text-primary"
            aria-hidden="true"
          />
          <Combobox.Input
            className="h-12 w-full border-0 bg-transparent pr-4 pl-11 text-13 text-primary outline-none placeholder:text-placeholder focus:ring-0"
            placeholder={t("common.search.placeholder")}
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            tabIndex={baseTabIndex}
          />
        </div>

        <div className="flex flex-col-reverse gap-4 p-2 text-13 text-secondary sm:flex-row sm:items-center sm:justify-between">
          {selectedIssues.length > 0 ? (
            <div className="mt-1 flex flex-wrap items-center gap-2">
              {selectedIssues.map((issue) => (
                <div
                  key={issue.id}
                  className="flex items-center gap-1 rounded-md border border-subtle bg-layer-1 py-1 pl-2 text-11 whitespace-nowrap text-primary"
                >
                  <IssueIdentifier
                    projectId={issue.project_id}
                    issueTypeId={issue.type_id}
                    projectIdentifier={issue.project__identifier}
                    issueSequenceId={issue.sequence_id}
                    size="xs"
                    variant="secondary"
                  />
                  <button
                    type="button"
                    className="group p-1"
                    disabled={isSubmitting}
                    aria-label={`Remove ${issue.name}`}
                    onClick={() =>
                      form.setFieldValue("selectedIssues", (items) => items.filter((item) => item.id !== issue.id))
                    }
                  >
                    <CloseIcon className="h-3 w-3 text-secondary group-hover:text-primary" />
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <div className="w-min rounded-md border border-subtle bg-layer-1 p-2 text-11 whitespace-nowrap">
              {t("issue.select.empty")}
            </div>
          )}
          {workspaceLevelToggle && (
            <Tooltip tooltipContent="Toggle workspace level search" isMobile={isMobile}>
              <div
                className={`flex flex-shrink-0 cursor-pointer items-center gap-1 text-11 ${
                  isWorkspaceLevel ? "text-primary" : "text-secondary"
                }`}
              >
                <ToggleSwitch
                  label="Workspace search"
                  disabled={isSubmitting}
                  value={isWorkspaceLevel}
                  onChange={() => {
                    if (!isSubmitting) setIsWorkspaceLevel((previous) => !previous);
                  }}
                />
                <button
                  type="button"
                  disabled={isSubmitting}
                  onClick={() => setIsWorkspaceLevel((previous) => !previous)}
                  className="flex-shrink-0"
                >
                  {t("common.workspace_level")}
                </button>
              </div>
            </Tooltip>
          )}
        </div>

        <Combobox.Options
          modal={false}
          static
          className="vertical-scrollbar scrollbar-md max-h-80 scroll-py-2 overflow-y-auto"
        >
          <IssueSelectionResults
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            query={search}
            issues={filteredIssues}
            selectedIssues={selectedIssues}
            isSearching={isSearching}
            searchTerm={searchTerm}
            debouncedSearchTerm={debouncedSearchTerm}
          />
        </Combobox.Options>
      </Combobox>
      {errors.map((error) => (
        <p role="alert" key={error} className="px-3 text-13 text-danger-primary">
          {error}
        </p>
      ))}
      <div className="flex items-center justify-between p-3">
        <Button
          type="button"
          variant="link"
          onClick={handleSelectIssues}
          disabled={filteredIssues.length === 0 || isSearching || isSubmitting || search.isError}
          className="p-0"
        >
          {allVisibleSelected ? t("issue.select.deselect_all") : t("issue.select.select_all")}
        </Button>
        <div className="flex items-center justify-end gap-2">
          <Button type="button" variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
            {t("common.cancel")}
          </Button>
          <Button
            type="button"
            variant="primary"
            size="lg"
            onClick={onSubmit}
            loading={isSubmitting}
            disabled={isSubmitting || selectedIssues.length === 0 || !search.accessReady}
          >
            {isSubmitting ? t("common.adding") : t("issue.select.add_selected")}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}

function IssueSelectionResults({
  workspaceSlug,
  projectId,
  query,
  issues,
  selectedIssues,
  isSearching,
  searchTerm,
  debouncedSearchTerm,
}: {
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  query: ReturnType<typeof useProjectIssueSearch>;
  issues: ISearchIssueResponse[];
  selectedIssues: ISearchIssueResponse[];
  isSearching: boolean;
  searchTerm: string;
  debouncedSearchTerm: string;
}) {
  if (!workspaceSlug || !projectId)
    return (
      <p role="alert" className="p-3 text-danger-primary">
        A workspace and project are required to search work items.
      </p>
    );
  if (query.isError)
    return (
      <div role="alert" className="p-3 text-danger-primary">
        Could not search work items.
        <Button type="button" variant="link" onClick={() => void query.retry()}>
          Retry search
        </Button>
      </div>
    );
  if (isSearching)
    return (
      <Loader className="space-y-3 p-3">
        <Loader.Item height="40px" />
        <Loader.Item height="40px" />
        <Loader.Item height="40px" />
        <Loader.Item height="40px" />
      </Loader>
    );
  if (!issues.length)
    return (
      <IssueSearchModalEmptyState
        debouncedSearchTerm={debouncedSearchTerm}
        isSearching={false}
        issues={issues}
        searchTerm={searchTerm}
      />
    );
  return (
    <ul className="p-2 text-13 text-primary">
      {issues.map((issue) => {
        const selected = selectedIssues.some((i) => i.id === issue.id);

        return (
          <Combobox.Option
            key={issue.id}
            as="div"
            value={issue}
            className={({ active }) =>
              `group my-0.5 flex w-full cursor-pointer items-center justify-between gap-2 rounded-md px-3 py-2 text-secondary select-none ${
                active ? "bg-layer-1 text-primary" : ""
              } ${selected ? "text-primary" : ""}`
            }
          >
            <div className="flex items-center gap-2 truncate">
              <input type="checkbox" aria-hidden="true" tabIndex={-1} checked={selected} readOnly />
              <span
                className="block h-1.5 w-1.5 flex-shrink-0 rounded-full"
                style={{
                  backgroundColor: issue.state__color,
                }}
              />
              <span className="flex-shrink-0">
                <IssueIdentifier
                  projectId={issue.project_id}
                  issueTypeId={issue.type_id}
                  projectIdentifier={issue.project__identifier}
                  issueSequenceId={issue.sequence_id}
                  size="xs"
                  variant="secondary"
                />
              </span>
              <span className="truncate">{issue.name}</span>
            </div>
            <a
              href={generateWorkItemLink({
                workspaceSlug,
                projectId: issue.project_id,
                issueId: issue.id,
                projectIdentifier: issue.project__identifier,
                sequenceId: issue.sequence_id,
              })}
              target="_blank"
              className="relative z-1 hidden flex-shrink-0 text-secondary group-hover:block hover:text-primary"
              rel="noopener noreferrer"
              onMouseDown={(event) => event.stopPropagation()}
              onClick={(event) => event.stopPropagation()}
            >
              <Rocket className="h-4 w-4" />
            </a>
          </Combobox.Option>
        );
      })}
    </ul>
  );
}
