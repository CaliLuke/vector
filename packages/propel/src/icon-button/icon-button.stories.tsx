/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Plus } from "lucide-react";
import { ExampleGroup, ExampleSizes } from "../storybook/example-gallery";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { IconButton } from "./icon-button";

const icon = () => <Plus size={16} />;

const meta = {
  title: "Components/IconButton",
  component: IconButton,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  args: {
    icon,
  },
} satisfies Meta<typeof IconButton>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    icon,
  },
};

export const Primary: Story = {
  args: {
    variant: "primary",
    icon,
  },
};

export const ErrorFill: Story = {
  args: {
    variant: "error-fill",
    icon,
  },
};

export const ErrorOutline: Story = {
  args: {
    variant: "error-outline",
    icon,
  },
};

export const Secondary: Story = {
  args: {
    variant: "secondary",
    icon,
  },
};

export const Tertiary: Story = {
  args: {
    variant: "tertiary",
    icon,
  },
};

export const Ghost: Story = {
  args: {
    variant: "ghost",
    icon,
  },
};

export const Small: Story = {
  args: {
    size: "sm",
    icon,
  },
};

export const Base: Story = {
  args: {
    size: "base",
    icon,
  },
};

export const Large: Story = {
  args: {
    size: "lg",
    icon,
  },
};

export const ExtraLarge: Story = {
  args: {
    size: "xl",
    icon,
  },
};

export const Loading: Story = {
  args: {
    loading: true,
    icon,
  },
};

export const Disabled: Story = {
  args: {
    disabled: true,
    icon,
  },
};

export const AllVariants: Story = {
  render() {
    return (
      <ExampleGroup title="Primary Variants">
        <IconButton variant="primary" icon={icon} />
        <IconButton variant="error-fill" icon={icon} />
        <IconButton variant="error-outline" icon={icon} />
        <IconButton variant="secondary" icon={icon} />
        <IconButton variant="tertiary" icon={icon} />
        <IconButton variant="ghost" icon={icon} />
      </ExampleGroup>
    );
  },
};

export const AllSizes: Story = {
  render() {
    return (
      <ExampleSizes>
        <IconButton size="sm" icon={icon} />
        <IconButton size="base" icon={icon} />
        <IconButton size="lg" icon={icon} />
        <IconButton size="xl" icon={icon} />
      </ExampleSizes>
    );
  },
};

export const AllStates: Story = {
  render() {
    return (
      <ExampleGroup title="Button States">
        <IconButton icon={icon} />
        <IconButton loading icon={icon} />
        <IconButton disabled icon={icon} />
      </ExampleGroup>
    );
  },
};
