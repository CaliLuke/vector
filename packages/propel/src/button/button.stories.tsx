/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { prependIconExample, appendIconExample } from "../storybook/icon-example-args";
import { ExampleGroup, ExampleSizes } from "../storybook/example-gallery";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button } from "./button";

const meta = {
  title: "Components/Button",
  component: Button,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  args: {
    children: "Button",
  },
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Primary: Story = {
  args: {
    variant: "primary",
    children: "Primary Button",
  },
};

export const ErrorFill: Story = {
  args: {
    variant: "error-fill",
    children: "Error Button",
  },
};

export const ErrorOutline: Story = {
  args: {
    variant: "error-outline",
    children: "Error Outline Button",
  },
};

export const Secondary: Story = {
  args: {
    variant: "secondary",
    children: "Secondary Button",
  },
};

export const Tertiary: Story = {
  args: {
    variant: "tertiary",
    children: "Tertiary Button",
  },
};

export const Ghost: Story = {
  args: {
    variant: "ghost",
    children: "Ghost Button",
  },
};

export const Link: Story = {
  args: {
    variant: "link",
    children: "Link Button",
  },
};

export const Small: Story = {
  args: {
    size: "sm",
    children: "Small Button",
  },
};

export const Base: Story = {
  args: {
    size: "base",
    children: "Base Button",
  },
};

export const Large: Story = {
  args: {
    size: "lg",
    children: "Large Button",
  },
};

export const ExtraLarge: Story = {
  args: {
    size: "xl",
    children: "Extra Large Button",
  },
};

export const Loading: Story = {
  args: {
    loading: true,
    children: "Loading Button",
  },
};

export const Disabled: Story = {
  args: {
    disabled: true,
    children: "Disabled Button",
  },
};

export const WithPrependIcon: Story = prependIconExample;

export const WithAppendIcon: Story = appendIconExample;

export const AllVariants: Story = {
  render() {
    return (
      <ExampleGroup title="Primary Variants">
        <Button variant="primary">Primary</Button>
        <Button variant="error-fill">Error Fill</Button>
        <Button variant="error-outline">Error Outline</Button>
        <Button variant="secondary">Secondary</Button>
        <Button variant="tertiary">Tertiary</Button>
        <Button variant="ghost">Ghost</Button>
        <Button variant="link">Link</Button>
      </ExampleGroup>
    );
  },
};

export const AllSizes: Story = {
  render() {
    return (
      <ExampleSizes>
        <Button size="sm">Small</Button>
        <Button size="base">Base</Button>
        <Button size="lg">Large</Button>
        <Button size="xl">Extra Large</Button>
      </ExampleSizes>
    );
  },
};

export const AllStates: Story = {
  render() {
    return (
      <ExampleGroup title="Button States">
        <Button>Default</Button>
        <Button loading>Loading</Button>
        <Button disabled>Disabled</Button>
      </ExampleGroup>
    );
  },
};
