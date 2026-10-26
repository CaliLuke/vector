/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Logo } from "@plane/propel/emoji-icon-picker";
import { CloseIcon } from "@plane/propel/icons";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

type Props = {
  handleRemove: (val: string) => void;
  values: string[];
  editable: boolean | undefined;
};

export function AppliedProjectFilters(props: Props) {
  const { handleRemove, values, editable } = props;
  // store hooks
  const projectRows = useWorkspaceProjectDetails().projects;

  return (
    <>
      {values.map((projectId) => {
        const projectDetails = projectRows?.find((project) => project.id === projectId) ?? null;

        if (!projectDetails) return null;

        return (
          <div key={projectId} className="flex items-center gap-1 rounded-sm bg-layer-1 p-1 text-11">
            <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
              <Logo logo={projectDetails.logo_props} size={12} />
            </span>
            <span className="normal-case">{projectDetails.name}</span>
            {editable && (
              <button
                type="button"
                className="grid place-items-center text-tertiary hover:text-secondary"
                onClick={() => handleRemove(projectId)}
              >
                <CloseIcon height={10} width={10} strokeWidth={2} />
              </button>
            )}
          </div>
        );
      })}
    </>
  );
}
