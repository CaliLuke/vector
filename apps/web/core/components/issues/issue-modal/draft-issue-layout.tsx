/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { workItemSchemaService } from "@/services/work-item-schema.service";
import { submittedPropertyValues } from "@/components/work-item-types/item-property-draft";
import { useState } from "react";
import isEmpty from "lodash-es/isEmpty.js";

// Vector imports
import type { TIssue } from "@plane/types";
import { isEmptyHtmlString } from "@plane/utils/string";
// hooks
// local imports
import { ConfirmIssueDiscard } from "../confirm-issue-discard";
import { itemAssetReferences } from "./item-asset-references";
import { IssueFormRoot } from "./form";
import type { IssueFormProps } from "./form";

export interface DraftIssueProps extends IssueFormProps {
  changesMade: Partial<TIssue> | null;
  onChange: (formData: Partial<TIssue> | null) => void;
}

export const DraftIssueLayout = function DraftIssueLayout(props: DraftIssueProps) {
  const { changesMade, data, onChange, onClose, onSubmit, projectId } = props;
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // states
  const [issueDiscardModal, setIssueDiscardModal] = useState(false);
  const sanitizeChanges = (): Partial<TIssue> => {
    const sanitizedChanges = { ...changesMade };
    Object.entries(sanitizedChanges).forEach(([key, value]) => {
      const issueKey = key as keyof TIssue;
      if (value === null || value === undefined || value === "") delete sanitizedChanges[issueKey];
      if (typeof value === "object" && isEmpty(value)) delete sanitizedChanges[issueKey];
      if (Array.isArray(value) && value.length === 0) delete sanitizedChanges[issueKey];
      if (issueKey === "project_id") delete sanitizedChanges.project_id;
      if (issueKey === "priority" && value && value === "none") delete sanitizedChanges.priority;
      if (
        issueKey === "description_html" &&
        changesMade?.description_html &&
        isEmptyHtmlString(changesMade.description_html, ["img"])
      )
        delete sanitizedChanges.description_html;
    });
    return sanitizedChanges;
  };

  const handleClose = () => {
    // If the user is updating an existing work item, we don't need to show the discard modal
    if (data?.id) {
      onClose();
      setIssueDiscardModal(false);
    } else {
      if (changesMade) {
        const sanitizedChanges = sanitizeChanges();
        if (isEmpty(sanitizedChanges)) {
          onClose();
          setIssueDiscardModal(false);
        } else setIssueDiscardModal(true);
      } else {
        onClose();
        setIssueDiscardModal(false);
      }
    }
  };

  const handleCreateDraftIssue = async () => {
    if (!changesMade) return;
    const targetProject = changesMade.project_id !== undefined ? changesMade.project_id : projectId;
    const schema = await workItemSchemaService.schema(workspaceSlug, {
      mode: "draft",
      project_id: targetProject || undefined,
      type_id: changesMade.type_id ?? "",
    });
    await onSubmit(
      {
        ...changesMade,
        name: changesMade.name?.trim() ?? "",
        project_id: targetProject,
        custom_properties: submittedPropertyValues(changesMade.custom_properties ?? {}, schema),
      },
      true,
      itemAssetReferences(changesMade, schema),
      schema.revision
    );
    onChange(null);
    setIssueDiscardModal(false);
  };

  return (
    <>
      <ConfirmIssueDiscard
        isOpen={issueDiscardModal}
        handleClose={() => setIssueDiscardModal(false)}
        onConfirm={handleCreateDraftIssue}
        onDiscard={() => {
          onChange(null);
          setIssueDiscardModal(false);
          onClose();
        }}
      />
      <IssueFormRoot {...props} onClose={handleClose} />
    </>
  );
};
