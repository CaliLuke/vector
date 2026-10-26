import { useParams } from "@tanstack/react-router";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { ETabIndices } from "@plane/constants";
import { getTabIndex } from "@plane/utils/tab-indices";
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
import { IntakeStateDropdown } from "@/components/dropdowns/intake-state/dropdown";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { DateDropdown } from "@/components/dropdowns/date";
import { IssueLabelSelect } from "@/components/issues/select";
import type { IssueDraftForm } from "@/components/issues/issue-modal/form-state";
import { usePlatformOS } from "@/hooks/use-platform-os";
export function InboxIssueProperties({ projectId, form }: { projectId: string; form: IssueDraftForm }) {
  const { workspaceSlug } = useParams({ strict: false });
  const { isMobile } = usePlatformOS();
  const { getIndex } = getTabIndex(ETabIndices.INTAKE_ISSUE_FORM, isMobile);
  return (
    <div className="relative flex flex-wrap items-center gap-2">
      <form.Field name="state_id">
        {(field) => (
          <div className="h-7">
            <IntakeStateDropdown
              value={field.state.value}
              onChange={field.handleChange}
              projectId={projectId}
              buttonVariant="border-with-text"
              tabIndex={getIndex("state_id")}
            />
          </div>
        )}
      </form.Field>
      <form.Field name="priority">
        {(field) => (
          <div className="h-7">
            <PriorityDropdown
              value={field.state.value}
              onChange={field.handleChange}
              buttonVariant="border-with-text"
              tabIndex={getIndex("priority")}
            />
          </div>
        )}
      </form.Field>
      <form.Field name="assignee_ids">
        {(field) => (
          <div className="h-7">
            <MemberDropdown
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              value={field.state.value}
              onChange={field.handleChange}
              buttonVariant={field.state.value.length ? "transparent-without-text" : "border-with-text"}
              buttonClassName={field.state.value.length ? "hover:bg-transparent" : ""}
              placeholder="Assignees"
              multiple
              tabIndex={getIndex("assignee_ids")}
            />
          </div>
        )}
      </form.Field>
      <form.Field name="label_ids">
        {(field) => (
          <div className="h-7">
            <IssueLabelSelect
              projectId={projectId}
              value={field.state.value}
              onChange={field.handleChange}
              tabIndex={getIndex("label_ids")}
            />
          </div>
        )}
      </form.Field>
      <form.Field name="target_date">
        {(field) => (
          <div className="h-7">
            <DateDropdown
              value={field.state.value}
              onChange={(date) => field.handleChange(renderFormattedPayloadDate(date) ?? null)}
              buttonVariant="border-with-text"
              placeholder="Due date"
              tabIndex={getIndex("target_date")}
            />
          </div>
        )}
      </form.Field>
    </div>
  );
}
