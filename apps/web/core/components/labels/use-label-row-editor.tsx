/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import type { Dispatch, SetStateAction } from "react";
import type { IIssueLabel } from "@plane/types";
import { CreateUpdateLabelInline } from "./create-update-label-inline";
import type { TLabelOperationsCallbacks } from "./create-update-label-inline";

type Options = {
  label: IIssueLabel;
  setIsUpdating: Dispatch<SetStateAction<boolean>>;
  labelOperationsCallbacks: TLabelOperationsCallbacks;
};

export function useLabelRowEditor({ label, setIsUpdating, labelOperationsCallbacks }: Options) {
  const [isEditing, setEditing] = useState(false);
  const edit = () => {
    setEditing(true);
    setIsUpdating(true);
  };
  const editor = isEditing ? (
    <CreateUpdateLabelInline
      labelForm={isEditing}
      setLabelForm={setEditing}
      isUpdating
      labelToUpdate={label}
      labelOperationsCallbacks={labelOperationsCallbacks}
      onClose={() => {
        setEditing(false);
        setIsUpdating(false);
      }}
    />
  ) : null;
  return { editor, edit };
}
