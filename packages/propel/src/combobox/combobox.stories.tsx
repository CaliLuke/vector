/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChevronsUpDown } from "lucide-react";
import { useArgs } from "storybook/preview-api";
import { CheckIcon } from "../icons";
import { Combobox } from "./combobox";
import type { ComboboxProps, ComboboxOptionsProps } from "./combobox";

const frameworks = [
  { value: "react", label: "React" },
  { value: "vue", label: "Vue" },
  { value: "angular", label: "Angular" },
  { value: "svelte", label: "Svelte" },
  { value: "solid", label: "Solid" },
  { value: "next", label: "Next.js" },
  { value: "nuxt", label: "Nuxt" },
  { value: "remix", label: "Remix" },
];

type DemoProps = ComboboxProps &
  Pick<ComboboxOptionsProps, "showSearch" | "maxHeight" | "emptyMessage"> & {
    disabledOptions?: string[];
  };

function selectionLabel(value: string | string[], limit?: number) {
  if (Array.isArray(value)) {
    if (value.length) return limit ? `${value.length}/${limit} selected` : `${value.length} selected`;
    return limit ? `Select up to ${limit} frameworks...` : "Select frameworks...";
  }
  return value ? frameworks.find((framework) => framework.value === value)?.label : "Select framework...";
}

function FrameworkCombobox({ showSearch, maxHeight, emptyMessage, disabledOptions, ...props }: DemoProps) {
  const value = props.value ?? "";
  return (
    <div className="space-y-2">
      <Combobox {...props}>
        <Combobox.Button
          className={
            props.disabled
              ? "border-gray-300 bg-gray-100 flex w-72 items-center justify-between rounded-md border px-4 py-2 opacity-50"
              : "border-gray-300 hover:bg-gray-50 flex w-72 items-center justify-between rounded-md border bg-white px-4 py-2"
          }
        >
          <span className={props.multiSelect ? "truncate" : undefined}>
            {selectionLabel(value, props.maxSelections)}
          </span>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Combobox.Button>
        <Combobox.Options
          showSearch={showSearch}
          searchPlaceholder="Search framework..."
          maxHeight={maxHeight}
          emptyMessage={emptyMessage}
          className="w-72"
        >
          {frameworks.map((framework) => (
            <Combobox.Option
              key={framework.value}
              value={framework.value}
              disabled={disabledOptions?.includes(framework.value)}
              className="flex items-center gap-2 px-4 py-2"
            >
              {(Array.isArray(value) ? value.includes(framework.value) : value === framework.value) && (
                <CheckIcon className="h-4 w-4" />
              )}
              <span>{framework.label}</span>
            </Combobox.Option>
          ))}
        </Combobox.Options>
      </Combobox>
      {props.maxSelections !== undefined && (
        <p className="text-gray-500 text-11">Maximum {props.maxSelections} selections allowed</p>
      )}
    </div>
  );
}

const meta = {
  title: "Components/Combobox",
  component: Combobox,
  subcomponents: {
    ComboboxButton: Combobox.Button,
    ComboboxOptions: Combobox.Options,
    ComboboxOption: Combobox.Option,
  },
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  args: { children: null, value: "", onValueChange: () => {}, showSearch: true },
  render(args) {
    const [{ value }, updateArgs] = useArgs<DemoProps>();
    return <FrameworkCombobox {...args} value={value} onValueChange={(next) => updateArgs({ value: next })} />;
  },
} satisfies Meta<DemoProps>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
export const WithoutSearch: Story = { args: { showSearch: false } };
export const MultiSelect: Story = { args: { multiSelect: true, value: [] } };
export const MultiSelectWithLimit: Story = { args: { multiSelect: true, value: [], maxSelections: 3 } };
export const Disabled: Story = { args: { disabled: true } };
export const DisabledOptions: Story = { args: { disabledOptions: ["angular", "svelte"] } };
export const CustomMaxHeight: Story = { args: { maxHeight: "sm" } };
export const CustomEmptyMessage: Story = { args: { emptyMessage: "No frameworks found. Try a different search." } };
