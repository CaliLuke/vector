/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import type { IInstanceTelemetryExporter } from "@plane/types";
import { instanceTelemetryOptions } from "@/queries/instance";

const statusLabels: Record<IInstanceTelemetryExporter["status"], string> = {
  enabled: "Enabled",
  disabled: "Disabled",
  not_configured: "Not configured",
  invalid_configuration: "Invalid configuration",
};

export function TelemetryStatus() {
  const telemetry = useQuery(instanceTelemetryOptions);

  return (
    <section aria-labelledby="telemetry-heading" className="space-y-4">
      <h2 id="telemetry-heading" className="border-b border-subtle pb-1.5 text-16 font-medium text-primary">
        Telemetry
      </h2>
      <p className="text-13 text-tertiary">
        Status and destination servers from your instance configuration. Enabled means sending is configured; it does
        not confirm delivery. Changes are managed on the server.
      </p>
      {telemetry.isError ? (
        <div role="alert" className="text-13 text-primary">
          Unable to load telemetry status.{" "}
          <Button type="button" variant="secondary" size="sm" onClick={() => void telemetry.refetch()}>
            Retry
          </Button>
        </div>
      ) : telemetry.data ? (
        <dl className="divide-y divide-subtle">
          {[
            { label: "Logs and traces", exporter: telemetry.data.logs_and_traces },
            { label: "Usage metrics", exporter: telemetry.data.instance_metrics },
            { label: "Analytics events", exporter: telemetry.data.analytics_events },
          ].map(({ label, exporter }) => (
            <div key={label} className="flex flex-col gap-2 py-3 sm:flex-row sm:justify-between sm:gap-6">
              <dt className="text-13 font-medium text-primary">{label}</dt>
              <dd className="min-w-0 text-13 sm:text-right">
                <span className="font-medium text-primary">{statusLabels[exporter.status]}</span>
                <div className="mt-1 break-all text-tertiary">
                  {exporter.destination
                    ? `${exporter.status === "enabled" ? "Destination" : "Configured destination"}: ${exporter.destination}`
                    : exporter.status === "invalid_configuration"
                      ? "Destination unavailable"
                      : "No destination configured"}
                </div>
              </dd>
            </div>
          ))}
        </dl>
      ) : (
        <output className="block text-13 text-tertiary">Loading telemetry status…</output>
      )}
    </section>
  );
}
