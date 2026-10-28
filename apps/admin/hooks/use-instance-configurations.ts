import { useMemo } from "react";
import { useRecordCollection } from "@plane/hooks";
import { instanceConfigurationsOptions } from "@/queries/instance";

export function useInstanceConfigurations() {
  const { select, ...options } = instanceConfigurationsOptions;
  const records = useRecordCollection(options, (row) => row.key);
  const data = useMemo(() => (records.data ? select?.(records.data) : undefined), [records.data, select]);
  return { ...records, data };
}
