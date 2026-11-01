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
  const handleClose = () => {
    if (!data?.id && changesMade && hasMeaningfulChanges(changesMade)) {
      setIssueDiscardModal(true);
    } else {
      onClose();
      setIssueDiscardModal(false);
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

function hasMeaningfulChanges(changes: Partial<TIssue>): boolean {
  return Object.entries(changes).some(([key, value]) => {
    if (key === "project_id" || value === null || value === undefined || value === "") return false;
    if (typeof value === "object" && isEmpty(value)) return false;
    if (key === "priority" && value === "none") return false;
    if (key === "description_html" && typeof value === "string" && isEmptyHtmlString(value, ["img"])) return false;
    return true;
  });
}
