import { useEffect, useState } from "react";
import type { IFilterOption, TSupportedFilterFieldConfigs, TFilterValue } from "@plane/types";

/** Options belong to the exact catalog config that requested them. */
export function useFilterSelectOptions<V extends TFilterValue>(config: TSupportedFilterFieldConfigs<V>) {
  const [result, setResult] = useState<{
    config: TSupportedFilterFieldConfigs<V>;
    options: IFilterOption<V>[];
    error: boolean;
  }>();
  useEffect(() => {
    let current = true;
    const load = async () => {
      try {
        const options =
          "getOptions" in config && typeof config.getOptions === "function" ? await config.getOptions() : [];
        if (current) setResult({ config, options, error: false });
      } catch {
        if (current) setResult({ config, options: [], error: true });
      }
    };
    void load();
    return () => {
      current = false;
    };
  }, [config]);
  const current = result?.config === config ? result : undefined;
  return { options: current?.options ?? [], loading: !current, error: current?.error ?? false };
}
