import { useEffect, useState } from "react";
import type { IFilterOption, TSupportedFilterFieldConfigs, TFilterValue } from "@plane/types";

const emptyOptions: never[] = [];

/** Options belong to the exact catalog config that requested them. */
export function useFilterSelectOptions<V extends TFilterValue>(config: TSupportedFilterFieldConfigs<V>) {
  const source = "getOptions" in config ? config.getOptions : emptyOptions;
  const [result, setResult] = useState<{
    config: TSupportedFilterFieldConfigs<V>;
    options: IFilterOption<V>[];
    error: boolean;
  }>();
  useEffect(() => {
    if (typeof source !== "function") return undefined;
    let current = true;
    const load = async () => {
      try {
        const options = await source();
        if (current) setResult({ config, options, error: false });
      } catch {
        if (current) setResult({ config, options: [], error: true });
      }
    };
    void load();
    return () => {
      current = false;
    };
  }, [config, source]);
  if (typeof source !== "function") return { options: source, loading: false, error: false };
  const current = result?.config === config ? result : undefined;
  return { options: current?.options ?? emptyOptions, loading: !current, error: current?.error ?? false };
}
