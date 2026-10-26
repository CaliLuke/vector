/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useForm, useStore } from "@tanstack/react-form";

import { Rocket } from "lucide-react";
import { Combobox } from "@headlessui/react";
// plane imports
import { useTranslation } from "@plane/i18n";
import type { ISearchIssueResponse, TProjectIssuesSearchParams } from "@plane/types";
import { generateWorkItemLink } from "@plane/utils/work-item";
import { SearchIcon } from "@plane/propel/icons";
import { Loader, EModalPosition, EModalWidth, ModalForm, FormErrors } from "@plane/ui";
// components
import { IssueSearchModalEmptyState } from "@/components/core/modals/issue-search-modal-empty-state";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import useDebounce from "@/hooks/use-debounce";
// services
import { useProjectIssueSearch } from "@/hooks/use-project-issue-search";

type Props = {
  workspaceSlug: string;
  projectId: string | undefined;
  issueId?: string;
  isOpen: boolean;
  selectedId?: string | null;
  filters?: Omit<TProjectIssuesSearchParams, "search" | "workspace_search">;
  onClose: () => void;
  onSelect: (issue: ISearchIssueResponse) => void | Promise<void>;
};

export function WorkItemSelectionModal(props: Props) {
  return props.isOpen ? (
    <WorkItemSelectionForm key={`${props.workspaceSlug}/${props.projectId}/${props.issueId}`} {...props} />
  ) : null;
}

function WorkItemSelectionForm({ workspaceSlug, projectId, issueId, onClose, onSelect, selectedId, filters }: Props) {
  // states
  const [query, setQuery] = useState("");
  const selection = useMutation({ mutationFn: async (issue: ISearchIssueResponse) => onSelect(issue), retry: false });
  const form = useForm({
    defaultValues: { issue: null as ISearchIssueResponse | null },
    onSubmit: async ({ value: draft }) => {
      if (!draft.issue) throw new Error("Select a work item.");
      if (!search.accessReady) throw new Error("Work item access changed. Retry the search.");
      form.setErrorMap({ onSubmit: undefined });
      try {
        await selection.mutateAsync(draft.issue, { onSuccess: () => onClose() });
      } catch (cause) {
        form.setErrorMap({ onSubmit: { form: "Could not save your selection. Please try again.", fields: {} } });
        throw cause;
      }
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const { t } = useTranslation();
  const debouncedSearchTerm = useDebounce(query, 500);

  const search = useProjectIssueSearch(workspaceSlug, projectId, {
    ...filters,
    search: debouncedSearchTerm,
    workspace_search: Boolean(filters?.parent || filters?.sub_issue),
  });
  const isSearching = search.isFetching || query !== debouncedSearchTerm;
  const filteredIssues = (search.data ?? []).filter((issue) => issue.id !== issueId);

  const issueList =
    filteredIssues.length > 0 ? (
      <li className="p-2">
        {query === "" && <h2 className="mt-4 mb-2 px-3 text-11 font-semibold text-primary">Select work item</h2>}
        <ul className="text-13 text-primary">
          {filteredIssues.map((issue) => {
            const stateColor = issue.state__color || "";

            return (
              <Combobox.Option
                key={issue.id}
                as="div"
                value={issue.id}
                className={({ active, selected }) =>
                  `group flex w-full cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-secondary select-none ${
                    active || selected ? "bg-layer-1 text-primary" : ""
                  } `
                }
              >
                <div className="flex items-center gap-2">
                  <span
                    className="block h-1.5 w-1.5 flex-shrink-0 rounded-full"
                    style={{
                      backgroundColor: stateColor,
                    }}
                  />
                  <IssueIdentifier
                    projectId={issue.project_id}
                    issueTypeId={issue.type_id}
                    projectIdentifier={issue.project__identifier}
                    issueSequenceId={issue.sequence_id}
                    size="xs"
                    variant="secondary"
                  />
                  <span className="text-secondary">{issue.name}</span>
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
                  rel="noopener noreferrer"
                  aria-label="Open work item in a new tab"
                  className="ml-auto hidden text-secondary group-hover:block hover:text-primary"
                  onMouseDown={(event) => event.stopPropagation()}
                  onClick={(event) => event.stopPropagation()}
                >
                  <Rocket className="h-4 w-4" />
                </a>
              </Combobox.Option>
            );
          })}
        </ul>
      </li>
    ) : (
      <IssueSearchModalEmptyState
        issues={filteredIssues}
        searchTerm={query}
        debouncedSearchTerm={debouncedSearchTerm}
        isSearching={isSearching}
      />
    );

  return (
    <ModalForm
      busy={busy}
      handleClose={onClose}
      onSubmit={() => {
        form.setErrorMap({ onSubmit: undefined });
        return form.handleSubmit().catch(() => {});
      }}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
      <fieldset disabled={busy} inert={busy}>
        <form.Field
          name="issue"
          validators={{
            onChange: ({ value: selected }) => (selected ? undefined : "Select a work item."),
            onSubmit: ({ value: selected }) => (selected ? undefined : "Select a work item."),
          }}
        >
          {(field) => (
            <>
              <Combobox
                value={field.state.value?.id ?? selectedId ?? ""}
                disabled={busy}
                onChange={(selected: string | null) => {
                  if (!selected || busy) return;
                  const issue = search.data?.find((item) => item.id === selected);
                  if (!issue) return;
                  field.handleChange(issue);
                  form.setErrorMap({ onSubmit: undefined });
                  void form.handleSubmit().catch(() => {});
                }}
              >
                <div className="relative m-1">
                  <SearchIcon
                    className="text-opacity-40 pointer-events-none absolute top-3.5 left-4 h-5 w-5 text-primary"
                    aria-hidden="true"
                  />
                  <Combobox.Input
                    type="text"
                    className="h-12 w-full border-0 bg-transparent pr-4 pl-11 text-primary outline-none focus:ring-0 sm:text-13"
                    placeholder={t("common.search.placeholder")}
                    value={query}
                    displayValue={() => query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                </div>

                <Combobox.Options
                  modal={false}
                  static
                  className="max-h-80 scroll-py-2 divide-y divide-subtle-1 overflow-y-auto"
                >
                  {search.isError ? (
                    <li className="p-3">
                      <FormErrors errors={["Could not load work items."]} />
                      <button
                        type="button"
                        onClick={() => {
                          void search.retry();
                        }}
                      >
                        Retry search
                      </button>
                    </li>
                  ) : isSearching ? (
                    <Loader className="space-y-3 p-3">
                      <Loader.Item height="40px" />
                      <Loader.Item height="40px" />
                      <Loader.Item height="40px" />
                      <Loader.Item height="40px" />
                    </Loader>
                  ) : (
                    <>{issueList}</>
                  )}
                </Combobox.Options>
              </Combobox>
              <FormErrors errors={[...field.state.meta.errors, error]} className="px-3 pb-3" />
              {error && (
                <button type="submit" className="px-3 pb-3 text-13 text-accent-primary">
                  Retry save
                </button>
              )}
            </>
          )}
        </form.Field>
      </fieldset>
    </ModalForm>
  );
}
