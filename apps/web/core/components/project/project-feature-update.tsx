/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";

import { useTranslation } from "@plane/i18n";
// ui
import { Button, getButtonStyling } from "@plane/propel/button";
import { Logo } from "@plane/propel/emoji-icon-picker";
import { Row } from "@plane/ui";
// components
// hooks
import { useProjectDetail } from "@/hooks/use-projects";
// plane web imports
import { ProjectFeaturesList } from "@/components/project/settings/features-list";

type Props = {
  workspaceSlug: string;
  projectId: string | null;
  onClose: () => void;
};

export function ProjectFeatureUpdate(props: Props) {
  const { workspaceSlug, projectId, onClose } = props;
  // store hooks
  const { t } = useTranslation();
  const projectQuery = useProjectDetail(workspaceSlug, projectId ?? undefined);

  if (!workspaceSlug || !projectId) return null;
  const currentProjectDetails = projectQuery.project;
  if (projectQuery.isError || projectQuery.accessError)
    return (
      <div role="alert">
        Project details could not be loaded.{" "}
        <button type="button" onClick={() => void projectQuery.retryAccess()}>
          Retry
        </button>
      </div>
    );
  if (projectQuery.accessDenied) return <div role="alert">You do not have access to this project.</div>;
  if (!currentProjectDetails) return null;

  return (
    <>
      <Row className="py-6">
        <ProjectFeaturesList workspaceSlug={workspaceSlug} projectId={projectId} isAdmin />
      </Row>
      <div className="mt-4 flex items-center justify-between gap-2 border-t border-subtle px-6 py-4">
        <div className="flex gap-1 text-13 font-medium text-tertiary">
          {t("congrats")}
          <Logo logo={currentProjectDetails.logo_props} /> <p className="break-all">{currentProjectDetails.name}</p>{" "}
          {t("created").toLowerCase()}.
        </div>
        <div className="flex gap-2">
          {/* eslint-disable-next-line jsx-a11y/tabindex-no-positive */}
          <Button variant="secondary" size="lg" onClick={onClose} tabIndex={1}>
            {t("close")}
          </Button>
          <Link
            to="/$workspaceSlug/projects/$projectId/issues/"
            params={{ workspaceSlug: workspaceSlug, projectId: projectId }}
            onClick={onClose}
            className={getButtonStyling("primary", "lg")}
            // oxlint-disable-next-line jsx-a11y/tabindex-no-positive
            tabIndex={2}
          >
            {t("open_project")}
          </Link>
        </div>
      </div>
    </>
  );
}
