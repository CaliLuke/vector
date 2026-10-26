/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Vector
import { Row } from "@plane/ui";
// components
import { cn } from "@plane/utils/common";
import { BLOCK_HEIGHT } from "@/components/gantt-chart/constants";
import { ModuleGanttSidebarBlock } from "@/components/modules";
// helpers
// hooks
import { useGanttChartState } from "../../contexts/chart-state";
import { daysFromGanttWidth } from "../../state";
import { GanttSidebarDuration } from "../common";

type Props = {
  blockId: string;
  isDragging: boolean;
};

export const ModulesSidebarBlock = function ModulesSidebarBlock(props: Props) {
  const { blockId, isDragging } = props;
  // store hooks
  const { blocksById, activeBlockId, setActiveBlockId, viewData } = useGanttChartState();
  const block = blocksById[blockId];

  if (!block) return <></>;

  const isBlockComplete = !!block.start_date && !!block.target_date;
  const duration = isBlockComplete ? daysFromGanttWidth(viewData, block?.position?.width) : undefined;

  return (
    <div
      className={cn({
        "rounded-sm bg-layer-1": isDragging,
      })}
      onMouseEnter={() => setActiveBlockId(block.id)}
      onMouseLeave={() => setActiveBlockId(null)}
    >
      <Row
        id={`sidebar-block-${block.id}`}
        className={cn(
          "group flex w-full items-center gap-2 bg-layer-transparent pr-4 hover:bg-layer-transparent-hover",
          {
            "bg-transparent-hover": activeBlockId === block.id,
          }
        )}
        style={{
          height: `${BLOCK_HEIGHT}px`,
        }}
      >
        <div className="flex h-full flex-grow items-center justify-between gap-2 truncate">
          <div className="flex-grow truncate">
            <ModuleGanttSidebarBlock module={block.data} />
          </div>
          {duration !== undefined && <GanttSidebarDuration days={duration} />}
        </div>
      </Row>
    </div>
  );
};
