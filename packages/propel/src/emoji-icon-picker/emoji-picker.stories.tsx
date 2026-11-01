/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import type { ComponentProps } from "react";
import { useArgs } from "storybook/preview-api";
import { useForm } from "@tanstack/react-form";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { EmojiPicker } from "./emoji-picker";
import type { TChangeHandlerProps } from "./helper";
import { EmojiIconPickerTypes } from "./helper";

type DemoProps = ComponentProps<typeof EmojiPicker> & {
  selectionView?: "details" | "emoji" | "icon" | "icon-details" | "compact" | "color";
};

function SelectionResult({ value, view }: { value: TChangeHandlerProps; view: DemoProps["selectionView"] }) {
  if (view === "emoji") return <div className="text-13">Selected: {value.type === "emoji" ? value.value : "Icon"}</div>;
  if (view === "icon")
    return (
      <div className="text-13">
        Selected: {value.type === "icon" && typeof value.value === "object" ? value.value.name : "Emoji"}
      </div>
    );
  if (view === "compact") return <div className="text-13">Selected: {JSON.stringify(value)}</div>;
  return (
    <div className="rounded-sm border border-subtle bg-layer-1 p-4 text-13">
      {view !== "color" && (
        <div className="mb-2 font-medium">{view === "icon-details" ? "Selected Icon:" : "Selected:"}</div>
      )}
      <pre className="text-11">{JSON.stringify(value, null, 2)}</pre>
    </div>
  );
}

const meta = {
  title: "Components/Emoji/EmojiPicker",
  component: EmojiPicker,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "😊 Pick an emoji or icon",
    defaultOpen: EmojiIconPickerTypes.EMOJI,
    closeOnSelect: true,
  },
  render({ selectionView, ...args }) {
    const [{ isOpen }, updateArgs] = useArgs<DemoProps>();
    const [selectedValue, setSelectedValue] = useState<TChangeHandlerProps | null>(null);
    return (
      <div className="space-y-4 p-4">
        <EmojiPicker
          {...args}
          isOpen={isOpen}
          handleToggle={(open) => updateArgs({ isOpen: open })}
          onChange={(value) => {
            setSelectedValue(value);
            args.onChange(value);
          }}
        />
        {selectedValue && <SelectionResult value={selectedValue} view={selectionView} />}
      </div>
    );
  },
} satisfies Meta<DemoProps>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
export const OpenToEmojiTab: Story = { args: { label: "😊 Choose Emoji", selectionView: "emoji" } };
export const OpenToIconTab: Story = {
  args: { label: "🎨 Choose Icon", defaultOpen: EmojiIconPickerTypes.ICON, selectionView: "icon" },
};
export const LucideIcons: Story = {
  args: {
    label: "Lucide Icons",
    defaultOpen: EmojiIconPickerTypes.ICON,
    iconType: "lucide",
    selectionView: "icon-details",
  },
};
export const MaterialIcons: Story = {
  args: {
    label: "Material Icons",
    defaultOpen: EmojiIconPickerTypes.ICON,
    iconType: "material",
    selectionView: "icon-details",
  },
};

export const CloseOnSelectDisabled: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "Close On Select Disabled",
  },
  render() {
    const [isOpen, setIsOpen] = useState(false);
    const [selectedValues, setSelectedValues] = useState<{ id: string; value: TChangeHandlerProps }[]>([]);

    const handleChange = (value: TChangeHandlerProps) => {
      setSelectedValues((prev) => [...prev, { id: crypto.randomUUID(), value }]);
    };

    return (
      <div className="space-y-4 p-4">
        <div className="flex items-center gap-2">
          <EmojiPicker
            isOpen={isOpen}
            handleToggle={setIsOpen}
            onChange={handleChange}
            label="Select Multiple (Stays Open)"
            defaultOpen={EmojiIconPickerTypes.EMOJI}
            closeOnSelect={false}
          />
          <button
            className="rounded-sm bg-layer-1 px-3 py-1.5 text-13 hover:bg-surface-2"
            onClick={() => setSelectedValues([])}
          >
            Clear
          </button>
        </div>
        {selectedValues.length > 0 && (
          <div className="rounded-sm border border-subtle bg-layer-1 p-4 text-13">
            <div className="mb-2 font-medium">Selected ({selectedValues.length}):</div>
            <div className="flex flex-wrap gap-2">
              {selectedValues.map(({ id, value }) => (
                <span key={id} className="text-16">
                  {value.type === "emoji" ? value.value : "🎨"}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    );
  },
};

export const CustomSearchPlaceholder: Story = {
  args: { label: "Custom Search", searchPlaceholder: "Type to find emojis...", selectionView: "compact" },
};
export const SearchDisabled: Story = { args: { label: "No Search", searchDisabled: true, selectionView: "compact" } };
export const CustomIconColor: Story = {
  args: {
    label: "Custom Icon Color",
    defaultOpen: EmojiIconPickerTypes.ICON,
    defaultIconColor: "#FF5733",
    selectionView: "color",
  },
};

function PlacementExample({ label, placement }: Pick<ComponentProps<typeof EmojiPicker>, "label" | "placement">) {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <div className="flex items-center gap-4">
      <span className="w-32 text-13">{label}:</span>
      <EmojiPicker isOpen={isOpen} handleToggle={setIsOpen} onChange={() => {}} label={label} placement={placement} />
    </div>
  );
}
export const DifferentPlacements: Story = {
  render: () => (
    <div className="space-y-8 p-8">
      <PlacementExample label="Bottom Start" placement="bottom-start" />
      <PlacementExample label="Bottom End" placement="bottom-end" />
      <PlacementExample label="Top Start" placement="top-start" />
      <PlacementExample label="Top End" placement="top-end" />
    </div>
  ),
};

export const InFormContext: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "In Form Context",
  },
  render() {
    const [isOpen, setIsOpen] = useState(false);
    const form = useForm({
      defaultValues: { title: "", emoji: null as TChangeHandlerProps | null },
      onSubmit: ({ value }) => {
        alert(`Form submitted:\n${JSON.stringify(value, null, 2)}`);
      },
    });

    return (
      <div className="max-w-md p-4">
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit();
          }}
          className="space-y-4 rounded-lg border border-subtle p-6"
        >
          <div>
            <label htmlFor="project-title" className="mb-2 block text-13 font-medium">
              Project Title
            </label>
            <form.Field name="title">
              {(field) => (
                <input
                  type="text"
                  id="project-title"
                  name={field.name}
                  onBlur={field.handleBlur}
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  className="w-full rounded-sm border border-subtle bg-layer-1 px-3 py-2"
                  placeholder="Enter project title"
                />
              )}
            </form.Field>
          </div>
          <div>
            <p className="mb-2 block text-13 font-medium">Project Icon</p>
            <form.Field name="emoji">
              {(field) => (
                <EmojiPicker
                  isOpen={isOpen}
                  handleToggle={setIsOpen}
                  onChange={field.handleChange}
                  label={
                    field.state.value && field.state.value.type === "emoji"
                      ? field.state.value.value
                      : "Click to select icon"
                  }
                  defaultOpen={EmojiIconPickerTypes.EMOJI}
                  closeOnSelect
                  buttonClassName="px-4 py-2 bg-layer-1 border border-subtle rounded-sm hover:bg-surface-2 w-full text-left"
                />
              )}
            </form.Field>
          </div>
          <button
            type="submit"
            className="w-full rounded-sm bg-accent-primary px-4 py-2 text-on-color hover:bg-accent-primary/80"
          >
            Create Project
          </button>
        </form>
      </div>
    );
  },
};
