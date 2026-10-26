/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import orderBy from "lodash-es/orderBy.js";
import { isTreeDropTargetData, getTreeDropInstruction as getInstructionFromPayload } from "@/lib/tree-drop-instruction";
import type { DragLocationHistory, DropTargetRecord } from "@atlaskit/pragmatic-drag-and-drop/types";
import type { ElementDragPayload } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { homeWidgetsOptions, updateHomeWidgetOptions } from "@/queries/home";
import { reorderedSortOrder } from "@/lib/reorder";
import { WidgetItem } from "./widget-item";

export const WidgetList = function WidgetList({ workspaceSlug }: { workspaceSlug: string }) {
  const { data = [] } = useQuery(homeWidgetsOptions(workspaceSlug));
  const widgets = orderBy(data, "sort_order", "desc");
  const update = useMutation(updateHomeWidgetOptions(useQueryClient(), workspaceSlug));
  const { t } = useTranslation();

  const handleDrop = (self: DropTargetRecord, source: ElementDragPayload, location: DragLocationHistory) => {
    if (update.isPending) return;
    const dropTargets = location?.current?.dropTargets ?? [];
    if (dropTargets.length <= 0) return;
    const dropTarget =
      dropTargets.length > 1 ? dropTargets.find((target: DropTargetRecord) => target?.data?.isChild) : dropTargets[0];

    const dropTargetData = dropTarget?.data;

    if (!dropTarget || !isTreeDropTargetData(dropTargetData)) return;
    const instruction = getInstructionFromPayload(dropTarget, source, location);
    const droppedId = dropTargetData.id;
    const sourceData = source.data;

    if (!isTreeDropTargetData(sourceData)) return;
    if (
      droppedId &&
      sourceData.id !== droppedId &&
      (instruction === "reorder-above" || instruction === "reorder-below")
    ) {
      const sort_order = reorderedSortOrder(
        widgets.map((widget) => ({ ...widget, id: widget.key })),
        sourceData.id,
        droppedId,
        instruction
      );
      update
        .mutateAsync({ key: sourceData.id, patch: { sort_order } })
        .then(() => {
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("toast.success"),
            message: t("home.widget.reordered_successfully"),
          });
        })
        .catch(() => {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: t("home.widget.reordering_failed"),
          });
        });
    }
  };

  return (
    <div className="my-4">
      {widgets.map((widget, index) => (
        <WidgetItem
          key={widget.key}
          widget={widget}
          isLastChild={index === widgets.length - 1}
          handleDrop={handleDrop}
          handleToggle={() => update.mutate({ key: widget.key, patch: { is_enabled: !widget.is_enabled } })}
          disabled={update.isPending}
        />
      ))}
    </div>
  );
};
