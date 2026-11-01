/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import {
  ActionsIconsMap,
  ArrowsIconsMap,
  LayoutIconsMap,
  MiscIconsMap,
  ProjectIconsMap,
  PropertyIconsMap,
  SubBrandIconsMap,
  WorkspaceIconsMap,
} from "./constants";
import { Icon } from "./icon";
import { CycleIcon } from "./project/cycle-icon";
import { HomeIcon } from "./workspace/home-icon";
import { ProjectIcon } from "./workspace/project-icon";

const meta = {
  title: "Icons",
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "A comprehensive collection of all icons used throughout the application. Supports both direct imports and registry-based usage.",
      },
    },
  },
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

const iconCategories = [
  { title: "Sub-Brand Icons", icons: SubBrandIconsMap },
  { title: "Workspace Icons", icons: WorkspaceIconsMap },
  { title: "Project Icons", icons: ProjectIconsMap },
  { title: "Layout Icons", icons: LayoutIconsMap },
  { title: "Property Icons", icons: PropertyIconsMap },
  { title: "Actions Icons", icons: ActionsIconsMap },
  { title: "Arrows Icons", icons: ArrowsIconsMap },
  { title: "Misc Icons", icons: MiscIconsMap, legacy: true },
];

function IconGrid({ items, legacy = false }: { items: { title: string; icon: React.ReactNode }[]; legacy?: boolean }) {
  return (
    <div className="grid w-full grid-cols-12 gap-6">
      {items.map((item) => (
        <div key={item.title} className="col-span-2 flex flex-col items-center justify-center gap-3 p-4">
          <div className={legacy ? "text-custom-text-200" : "text-secondary"}>{item.icon}</div>
          <p className={legacy ? "text-xs text-custom-text-300 text-center" : "text-center text-11 text-tertiary"}>
            {item.title}
          </p>
        </div>
      ))}
    </div>
  );
}

export const AllIcons: Story = {
  render: () => (
    <div className="space-y-12">
      {iconCategories.map((category) => (
        <div key={category.title} className="space-y-4">
          <h3
            className={
              category.legacy ? "text-lg text-custom-text-100 font-semibold" : "text-16 font-semibold text-primary"
            }
          >
            {category.title}
          </h3>
          <IconGrid items={category.icons} legacy={category.legacy} />
        </div>
      ))}
    </div>
  ),
};

export const RegistryUsage: Story = {
  render: () => (
    <div className="space-y-8">
      <div className="space-y-4">
        <h3 className="text-16 font-semibold text-primary">Registry-Based Usage</h3>
        <p className="text-13 text-tertiary">
          Use the <code className="rounded-sm bg-layer-1 px-1 py-0.5">Icon</code> component with{" "}
          <code className="rounded-sm bg-layer-1 px-1 py-0.5">name</code> prop for dynamic icon selection.
        </p>
        <IconGrid
          items={(["workspace.home", "project.cycle", "layout.board", "property.priority"] as const).map((name) => ({
            title: name,
            icon: <Icon name={name} className="text-secondary" />,
          }))}
        />
      </div>

      <div className="space-y-4">
        <h3 className="text-16 font-semibold text-primary">Direct Import Usage</h3>
        <p className="text-13 text-tertiary">
          Import icon components directly for better tree-shaking and type safety.
        </p>
        <IconGrid
          items={[
            { title: "HomeIcon", icon: <HomeIcon className="text-secondary" /> },
            { title: "CycleIcon", icon: <CycleIcon className="text-secondary" /> },
            { title: "ProjectIcon", icon: <ProjectIcon className="text-secondary" /> },
          ]}
        />
      </div>
    </div>
  ),
};

export const IconSizes: Story = {
  render: () => (
    <div className="space-y-6">
      <div className="space-y-4">
        <h3 className="text-16 font-semibold text-primary">Icon Sizes</h3>
        <p className="text-13 text-tertiary">Icons can be rendered in different sizes using width and height props.</p>
      </div>

      <div className="flex items-end gap-8">
        {[12, 16, 24, 32, 48].map((size) => (
          <div key={size} className="flex flex-col items-center gap-2">
            <Icon name="workspace.home" width={size} height={size} className="text-secondary" />
            <p className="text-11 text-tertiary">
              {size}x{size}
              {size === 16 ? " (default)" : ""}
            </p>
          </div>
        ))}
      </div>
    </div>
  ),
};
