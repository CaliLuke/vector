/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo, useState } from "react";
import { PolarGrid, Radar, RadarChart as CoreRadarChart, ResponsiveContainer, PolarAngleAxis } from "recharts";
import type { TRadarChartProps } from "@plane/types";
import { KeyedChartLegend } from "../components/legend";
import { CustomRadarAxisTick } from "../components/tick";
import { ChartTooltip } from "../components/tooltip";

function RadarChart<T extends string, K extends string>(props: TRadarChartProps<T, K>) {
  const { data, radars, margin, showTooltip, legend, className, angleAxis } = props;

  // states
  const [activeLegend, setActiveLegend] = useState<string | null>(null);

  const { itemKeys, itemLabels, itemDotColors } = useMemo(() => {
    const keys: string[] = [];
    const labels: Record<string, string> = {};
    const colors: Record<string, string> = {};

    for (const radar of radars) {
      keys.push(radar.key);
      labels[radar.key] = radar.name;
      colors[radar.key] = radar.stroke ?? radar.fill ?? "#000000";
    }
    return { itemKeys: keys, itemLabels: labels, itemDotColors: colors };
  }, [radars]);

  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <CoreRadarChart cx="50%" cy="50%" outerRadius="80%" data={data} margin={margin}>
          <PolarGrid stroke="var(--border-color-subtle)" />
          <PolarAngleAxis dataKey={angleAxis.key} tick={<CustomRadarAxisTick />} />
          {showTooltip && (
            <ChartTooltip
              activeKey={activeLegend}
              itemKeys={itemKeys}
              itemLabels={itemLabels}
              itemDotColors={itemDotColors}
            />
          )}
          {legend && <KeyedChartLegend options={legend} onActiveChange={setActiveLegend} />}
          {radars.map((radar) => (
            <Radar
              key={radar.key}
              name={radar.name}
              dataKey={radar.key}
              stroke={radar.stroke}
              fill={radar.fill}
              fillOpacity={radar.fillOpacity}
              dot={radar.dot}
            />
          ))}
        </CoreRadarChart>
      </ResponsiveContainer>
    </div>
  );
}

export { RadarChart };
