/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Dispatch, SetStateAction } from "react";
import { useEffect, useRef } from "react";

import { EditIcon, CloseIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
// types
import type { IIssueLabel } from "@plane/types";
// components
import type { TLabelOperationsCallbacks } from "./create-update-label-inline";
import { useLabelRowEditor } from "./use-label-row-editor";
import type { ICustomMenuItem } from "./label-block/label-item-block";
import { LabelItemBlock } from "./label-block/label-item-block";
import { LabelDndHOC } from "./label-drag-n-drop-HOC";

type Props = {
  label: IIssueLabel;
  handleLabelDelete: (label: IIssueLabel) => void;
  setIsUpdating: Dispatch<SetStateAction<boolean>>;
  isParentDragging?: boolean;
  isChild: boolean;
  isLastChild: boolean;
  onDrop: (
    draggingLabelId: string,
    droppedParentId: string | null,
    droppedLabelId: string | undefined,
    dropAtEndOfList: boolean
  ) => void;
  labelOperationsCallbacks: TLabelOperationsCallbacks;
  isEditable?: boolean;
};

export function ProjectSettingLabelItem(props: Props) {
  const {
    label,
    setIsUpdating,
    handleLabelDelete,
    isChild,
    isLastChild,
    isParentDragging = false,
    onDrop,
    labelOperationsCallbacks,
    isEditable = false,
  } = props;
  // states
  const { editor, edit } = useLabelRowEditor({ label, setIsUpdating, labelOperationsCallbacks });
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const { updateLabel } = labelOperationsCallbacks;

  const removeFromGroup = (selectedLabel: IIssueLabel) => {
    void updateLabel(selectedLabel.id, {
      parent: null,
    }).catch((error) => {
      if (!mounted.current) return;
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: error?.error ?? "Label could not be removed from its group. Please try again.",
      });
    });
  };

  const customMenuItems: ICustomMenuItem[] = [
    {
      CustomIcon: CloseIcon,
      onClick: removeFromGroup,
      isVisible: !!label.parent,
      text: "Remove from group",
      key: "remove_from_group",
    },
    {
      CustomIcon: EditIcon,
      onClick: edit,
      isVisible: true,
      text: "Edit label",
      key: "edit_label",
    },
  ];

  return (
    <LabelDndHOC label={label} isGroup={false} isChild={isChild} isLastChild={isLastChild} onDrop={onDrop}>
      {(isDragging, isDroppingInLabel, dragHandleRef) => (
        <div
          className={`rounded-sm ${isDroppingInLabel ? "border-[2px] border-accent-strong" : "border-[1.5px] border-transparent"}`}
        >
          <div
            className={`group relative flex items-center justify-between gap-2 space-y-3 rounded-sm px-1 py-3 ${
              isDroppingInLabel ? "" : "border-[0.5px] border-subtle"
            } ${isDragging || isParentDragging ? "bg-layer-1" : "bg-surface-1"}`}
          >
            {editor ?? (
              <LabelItemBlock
                label={label}
                isDragging={isDragging}
                customMenuItems={customMenuItems}
                handleLabelDelete={handleLabelDelete}
                dragHandleRef={dragHandleRef}
                disabled={!isEditable}
              />
            )}
          </div>
        </div>
      )}
    </LabelDndHOC>
  );
}
