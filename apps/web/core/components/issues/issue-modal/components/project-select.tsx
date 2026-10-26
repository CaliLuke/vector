/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDraftForm } from "../form-state";
import React from "react";

// plane imports
import { ETabIndices } from "@plane/constants";
// types
import { getTabIndex } from "@plane/utils/tab-indices";
// components
import { ProjectDropdown } from "@/components/dropdowns/project/dropdown";
// hooks
import { useIssueModal } from "@/hooks/context/use-issue-modal";
import { usePlatformOS } from "@/hooks/use-platform-os";

type TIssueProjectSelectProps = {
  form: IssueDraftForm;
  disabled?: boolean;
  allowEmpty?: boolean;
  onProjectChange: (projectId: string) => void;
  beforeChange: () => boolean;
};

export const IssueProjectSelect = function IssueProjectSelect(props: TIssueProjectSelectProps) {
  const { form, disabled = false, allowEmpty = false, onProjectChange, beforeChange } = props;
  // store hooks
  const { isMobile } = usePlatformOS();
  // context hooks
  const { allowedProjectIds } = useIssueModal();

  const { getIndex } = getTabIndex(ETabIndices.ISSUE_FORM, isMobile);

  const validate = ({ value }: { value: string | null }) => (allowEmpty || value ? undefined : "Select a project");
  return (
    <form.Field name="project_id" validators={{ onChange: validate, onSubmit: validate }}>
      {({ state: { value, meta }, handleChange: onChange }) => (
        <div className="min-w-0" data-issue-field="project_id" tabIndex={-1}>
          <div className="flex h-7 items-center gap-2">
            <ProjectDropdown
              value={value}
              onChange={(projectId) => {
                if (!beforeChange()) return;
                onChange(projectId);
                onProjectChange(projectId);
              }}
              multiple={false}
              buttonVariant="border-with-text"
              renderCondition={(projectId) => allowedProjectIds.includes(projectId)}
              tabIndex={getIndex("project_id")}
              disabled={disabled}
            />
            {allowEmpty && value && (
              <button
                type="button"
                disabled={disabled}
                className="text-12 text-secondary underline"
                onClick={() => {
                  if (beforeChange()) onChange(null);
                }}
              >
                No project
              </button>
            )}
          </div>
          {[...new Set(meta.errors)].map((error) => (
            <p key={error} role="alert" className="text-12 text-danger-primary">
              {error}
            </p>
          ))}
        </div>
      )}
    </form.Field>
  );
};
