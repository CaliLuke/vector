/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { useTranslation } from "@plane/i18n";

type Props = {
  feature: "module" | "page";
  assetPath: string;
  workspaceSlug: string;
  projectId: string;
  isManageDisabled: boolean;
};

export function ProjectFeatureDisabledState({ feature, assetPath, workspaceSlug, projectId, isManageDisabled }: Props) {
  const router = useRouter();
  const { t } = useTranslation();
  return (
    <div className="flex h-full w-full items-center justify-center">
      <DetailedEmptyState
        title={t(`disabled_project.empty_state.${feature}.title`)}
        description={t(`disabled_project.empty_state.${feature}.description`)}
        assetPath={assetPath}
        primaryButton={{
          text: t(`disabled_project.empty_state.${feature}.primary_button.text`),
          onClick: () => {
            void router.navigate({ href: `/${workspaceSlug}/settings/projects/${projectId}/features` });
          },
          disabled: isManageDisabled,
        }}
      />
    </div>
  );
}
