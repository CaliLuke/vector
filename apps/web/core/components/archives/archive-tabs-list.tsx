/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams, useLocation } from "@tanstack/react-router";

// types
import type { IProject } from "@plane/types";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

const ARCHIVES_TAB_LIST: {
  key: "issues" | "cycles" | "modules";
  label: string;
  shouldRender: (projectDetails: IProject) => boolean;
}[] = [
  {
    key: "issues",
    label: "Work items",
    shouldRender: () => true,
  },
  {
    key: "cycles",
    label: "Cycles",
    shouldRender: (projectDetails) => projectDetails.cycle_view,
  },
  {
    key: "modules",
    label: "Modules",
    shouldRender: (projectDetails) => projectDetails.module_view,
  },
];

export const ArchiveTabsList = function ArchiveTabsList() {
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { projectId } = useParams({ strict: false });
  const pathname = useLocation({ select: (location) => location.pathname });
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;

  // derived values
  if (!projectId) return null;
  const projectDetails = projectRows?.find((project) => project.id === projectId.toString());
  if (!projectDetails) return null;

  return (
    <>
      {ARCHIVES_TAB_LIST.map(
        (tab) =>
          tab.shouldRender(projectDetails) && (
            <Link
              key={tab.key}
              to={
                tab.key === "issues"
                  ? "/$workspaceSlug/projects/$projectId/archives/issues"
                  : tab.key === "cycles"
                    ? "/$workspaceSlug/projects/$projectId/archives/cycles"
                    : "/$workspaceSlug/projects/$projectId/archives/modules"
              }
              params={{ workspaceSlug, projectId }}
            >
              <span
                className={`flex min-w-min flex-shrink-0 border-b-2 px-4 py-4 text-13 font-medium whitespace-nowrap outline-none ${
                  pathname.includes(tab.key)
                    ? "border-accent-strong text-accent-primary"
                    : "border-transparent text-tertiary hover:border-subtle hover:text-placeholder"
                }`}
              >
                {tab.label}
              </span>
            </Link>
          )
      )}
    </>
  );
};
