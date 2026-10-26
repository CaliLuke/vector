/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";

import { useTheme } from "next-themes";
import { useForm, useStore } from "@tanstack/react-form";
import { Combobox } from "@headlessui/react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { SearchIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EIssuesStoreType } from "@plane/types";
import type { ISearchIssueResponse } from "@plane/types";
import { Loader, EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// assets
import darkIssuesAsset from "@/app/assets/empty-state/search/issues-dark.webp?url";
import lightIssuesAsset from "@/app/assets/empty-state/search/issues-light.webp?url";
import darkSearchAsset from "@/app/assets/empty-state/search/search-dark.webp?url";
import lightSearchAsset from "@/app/assets/empty-state/search/search-light.webp?url";
// components
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";
// hooks
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useProjectIssueSearch } from "@/hooks/use-project-issue-search";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import useDebounce from "@/hooks/use-debounce";
// local components
import { BulkDeleteIssuesModalItem } from "./bulk-delete-issues-modal-item";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function BulkDeleteIssuesModal(props: Props) {
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const client = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const generation = useIdentityGeneration(client);
  const scopeKey = JSON.stringify([workspaceSlug, projectId, currentUser?.id, generation]);
  return props.isOpen && workspaceSlug && projectId ? (
    <BulkDeleteIssuesDialog
      key={scopeKey}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      onClose={props.onClose}
    />
  ) : null;
}

function BulkDeleteIssuesDialog({
  onClose,
  workspaceSlug,
  projectId,
}: Pick<Props, "onClose"> & { workspaceSlug: string; projectId: string }) {
  // states
  const [query, setQuery] = useState("");
  // hooks
  const { removeIssues } = useIssuesActions(EIssuesStoreType.PROJECT);
  const mounted = useCommittedScope(`${workspaceSlug}/${projectId}`);
  const client = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const identity = captureIdentityScope(client, currentUser?.id);
  const isCurrent = () => mounted() && Boolean(identity) && isIdentityScopeCurrent(client, identity);
  // derived values
  const debouncedSearchTerm: string = useDebounce(query, 500);

  const search = useProjectIssueSearch(workspaceSlug, projectId, {
    search: debouncedSearchTerm,
    workspace_search: false,
  });
  const issues = search.data ?? [];
  const isSearching = search.isFetching || query !== debouncedSearchTerm;
  const searchFailed = search.isError;
  const form = useForm({
    defaultValues: { delete_issue_ids: [] as string[] },
    validators: {
      onSubmit: ({ value }): string | undefined =>
        value.delete_issue_ids.length ? undefined : "Select at least one work item.",
    },
    onSubmit: async ({ value }) => {
      if (!isCurrent() || !search.accessReady || searchFailed)
        throw new Error("Work item access changed. Retry the search.");
      await removeIssues(projectId, value.delete_issue_ids);
      if (!isCurrent()) return;
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Work items deleted successfully!" });
      onClose();
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const selectedIssues = useStore(form.store, (state) => state.values.delete_issue_ids);
  const errors = useStore(form.store, (state) => state.errors);
  const handleClose = () => {
    if (!form.state.isSubmitting) onClose();
  };

  return (
    <ModalCore isOpen handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          if (form.state.isSubmitting) return;
          void form.handleSubmit().catch(() => {
            if (isCurrent()) form.setErrorMap({ onSubmit: "Work items could not be deleted. Please try again." });
          });
        }}
      >
        <Combobox
          disabled={isSubmitting}
          onChange={(val: string | null) => {
            if (val === null || isSubmitting) return;
            if (selectedIssues.includes(val))
              form.setFieldValue(
                "delete_issue_ids",
                selectedIssues.filter((i) => i !== val)
              );
            else form.setFieldValue("delete_issue_ids", [...selectedIssues, val]);
          }}
        >
          <div className="relative m-1">
            <SearchIcon
              className="text-opacity-40 pointer-events-none absolute top-3.5 left-4 h-5 w-5 text-primary"
              aria-hidden="true"
            />
            <input
              type="text"
              className="h-12 w-full border-0 bg-transparent pr-4 pl-11 text-primary outline-none focus:ring-0 sm:text-13"
              placeholder="Search..."
              disabled={isSubmitting}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </div>

          <Combobox.Options
            modal={false}
            static
            className="max-h-80 scroll-py-2 divide-y divide-subtle-1 overflow-y-auto"
          >
            {searchFailed ? (
              <p role="alert" className="p-3 text-danger-primary">
                Unable to load work items.{" "}
                <button type="button" onClick={() => void search.retry()} disabled={isSubmitting}>
                  Retry search
                </button>
              </p>
            ) : isSearching ? (
              <Loader className="space-y-3 p-3">
                <Loader.Item height="40px" />
                <Loader.Item height="40px" />
                <Loader.Item height="40px" />
                <Loader.Item height="40px" />
              </Loader>
            ) : (
              <BulkDeleteSearchResults issues={issues} selectedIssues={selectedIssues} query={query} />
            )}
          </Combobox.Options>
        </Combobox>

        {errors.map((error) => (
          <p key={String(error)} role="alert" className="px-3 text-danger-primary">
            {String(error)}
          </p>
        ))}
        {(issues.length > 0 || selectedIssues.length > 0) && (
          <div className="flex items-center justify-end gap-2 p-3">
            <Button type="button" variant="secondary" size="lg" disabled={isSubmitting} onClick={handleClose}>
              Cancel
            </Button>
            <Button
              variant="error-fill"
              size="lg"
              type="submit"
              disabled={!selectedIssues.length || isSubmitting || !search.accessReady || searchFailed || isSearching}
              loading={isSubmitting}
            >
              {isSubmitting ? "Deleting..." : "Delete selected work items"}
            </Button>
          </div>
        )}
      </form>
    </ModalCore>
  );
}

function BulkDeleteSearchResults({
  issues,
  selectedIssues,
  query,
}: {
  issues: ISearchIssueResponse[];
  selectedIssues: string[];
  query: string;
}) {
  const { resolvedTheme } = useTheme();
  const { t } = useTranslation();
  const searchResolvedPath = resolvedTheme === "light" ? lightSearchAsset : darkSearchAsset;
  const issuesResolvedPath = resolvedTheme === "light" ? lightIssuesAsset : darkIssuesAsset;
  return issues.length > 0 ? (
    <li className="p-2">
      {query === "" && (
        <h2 className="mt-4 mb-2 px-3 text-11 font-semibold text-primary">Select work items to delete</h2>
      )}
      <ul className="text-13 text-secondary">
        {issues.map((issue) => (
          <BulkDeleteIssuesModalItem
            issue={issue}
            canDeleteIssueIds={selectedIssues.includes(issue.id)}
            key={issue.id}
          />
        ))}
      </ul>
    </li>
  ) : (
    <div className="flex flex-col items-center justify-center px-3 py-8 text-center">
      {query === "" ? (
        <SimpleEmptyState title={t("issue_relation.empty_state.no_issues.title")} assetPath={issuesResolvedPath} />
      ) : (
        <SimpleEmptyState title={t("issue_relation.empty_state.search.title")} assetPath={searchResolvedPath} />
      )}
    </div>
  );
}
