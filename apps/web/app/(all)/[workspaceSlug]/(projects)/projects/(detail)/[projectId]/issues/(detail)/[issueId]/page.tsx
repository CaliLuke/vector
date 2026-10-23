/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, redirect, isRedirect, useRouter } from "@tanstack/react-router";
import { useTheme } from "next-themes";

import { useTranslation } from "@plane/i18n";
// assets
import emptyIssueDark from "@/app/assets/empty-state/search/issues-dark.webp?url";
import emptyIssueLight from "@/app/assets/empty-state/search/issues-light.webp?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { LogoSpinner } from "@/components/common/logo-spinner";
// hooks

// services
import { IssueService } from "@/services/issue/issue.service";
// types

const issueService = new IssueService();

async function loadIssue({ params }: { params: { workspaceSlug: string; projectId: string; issueId: string } }) {
  const { workspaceSlug, projectId, issueId } = params;

  try {
    const data = await issueService.getIssueMetaFromURL(workspaceSlug, projectId, issueId);

    if (data) {
      throw redirect({
        to: "/$workspaceSlug/browse/$workItem/",
        params: { workspaceSlug, workItem: `${data.project_identifier}-${data.sequence_id}` },
        replace: true,
      });
    }

    return { error: true, workspaceSlug };
  } catch (error) {
    // If it's a redirect, rethrow it
    if (isRedirect(error)) {
      throw error;
    }
    // Otherwise return error state
    return { error: true, workspaceSlug };
  }
}

function IssueDetailsPage(): React.JSX.Element | null {
  const loaderData = Route.useLoaderData();
  const router = useRouter();
  const { t } = useTranslation();
  const { resolvedTheme } = useTheme();

  if (loaderData.error) {
    return (
      <div className="flex size-full items-center justify-center">
        <EmptyState
          image={resolvedTheme === "dark" ? emptyIssueDark : emptyIssueLight}
          title={t("issue.empty_state.issue_detail.title")}
          description={t("issue.empty_state.issue_detail.description")}
          primaryButton={{
            text: t("issue.empty_state.issue_detail.primary_button.text"),
            onClick: () => void router.navigate({ href: `/${loaderData.workspaceSlug}/workspace-views/all-issues/` }),
          }}
        />
      </div>
    );
  }

  return (
    <div className="flex size-full items-center justify-center">
      <LogoSpinner />
    </div>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/issues/$issueId")({
  component: IssueDetailsPage,
  loader: loadIssue,
});
