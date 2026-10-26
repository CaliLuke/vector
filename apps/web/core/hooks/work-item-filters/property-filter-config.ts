import { ListFilter } from "lucide-react";
import type { TFilterConfig, TOperatorConfigMap, TSupportedOperators, TWorkItemFilterProperty } from "@plane/types";
import { CORE_OPERATORS, EXTENDED_OPERATORS, FILTER_FIELD_TYPE } from "@plane/types";
import {
  createFilterConfig,
  getDatePickerConfig,
  getDateRangePickerConfig,
  getSingleSelectConfig,
  getMultiSelectConfig,
} from "@plane/utils/rich-filters";
import type { WorkItemPropertyIdentity } from "@/services/work-item-types.service";

const operators = Object.values({ ...CORE_OPERATORS, ...EXTENDED_OPERATORS });
const dateLabels: Partial<Record<TSupportedOperators, { operatorLabel: string; negOperatorLabel: string }>> = {
  lt: { operatorLabel: "before", negOperatorLabel: "on or after" },
  lte: { operatorLabel: "on or before", negOperatorLabel: "after" },
  gt: { operatorLabel: "after", negOperatorLabel: "on or before" },
  gte: { operatorLabel: "on or after", negOperatorLabel: "before" },
};

export function propertyFilterConfig(field: WorkItemPropertyIdentity): TFilterConfig<TWorkItemFilterProperty> {
  const family = field.kind === "Formula" ? field.result_kind : field.kind;
  const config: TOperatorConfigMap = new Map();
  const common = { isOperatorEnabled: true, allowNegative: true as const };
  const selection = {
    items: field.options,
    getId: (option: { id: string; name: string }) => option.id,
    getLabel: (option: { id: string; name: string }) => option.name,
    getValue: (option: { id: string; name: string }) => option.id,
  };
  for (const operator of operators) {
    if (!field.operators.includes(operator)) continue;
    if (operator === "isnull") {
      config.set(
        operator,
        getSingleSelectConfig(
          {
            items: [
              { id: "empty", name: "empty", value: true },
              { id: "not-empty", name: "not empty", value: false },
            ],
            getId: (option) => option.id,
            getLabel: (option) => option.name,
            getValue: (option) => option.value,
          },
          { isOperatorEnabled: true, defaultValue: true }
        )
      );
    } else if (family === "Text" || family === "URL") {
      config.set(operator, { type: FILTER_FIELD_TYPE.TEXT, ...common });
    } else if (family === "Number") {
      config.set(operator, {
        type: operator === "range" ? FILTER_FIELD_TYPE.NUMBER_RANGE : FILTER_FIELD_TYPE.NUMBER,
        ...common,
      });
    } else if (family === "Date") {
      const labels = dateLabels[operator] ?? {};
      config.set(
        operator,
        operator === "range" ? getDateRangePickerConfig(common) : getDatePickerConfig({ ...common, ...labels })
      );
    } else if (family === "Boolean") {
      config.set(
        operator,
        getSingleSelectConfig(
          {
            items: [true, false],
            getId: String,
            getLabel: (value) => (value ? "True" : "False"),
            getValue: (value) => value,
          },
          common
        )
      );
    } else if (operator === "in") {
      config.set(operator, getMultiSelectConfig(selection, { ...common, singleValueOperator: "in" }));
    } else {
      config.set(operator, getSingleSelectConfig(selection, common));
    }
  }
  return createFilterConfig({
    id: field.key,
    label: field.name,
    icon: ListFilter,
    isEnabled: config.size > 0,
    supportedOperatorConfigsMap: config,
  });
}
