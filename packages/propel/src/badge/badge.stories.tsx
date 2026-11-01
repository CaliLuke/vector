/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { prependIconExample, appendIconExample } from "../storybook/icon-example-args";
import { ExampleGroup, ExampleSizes } from "../storybook/example-gallery";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Badge } from "./badge";

const meta = {
  title: "Components/Badge",
  component: Badge,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  args: {
    children: "Badge",
  },
} satisfies Meta<typeof Badge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Neutral: Story = {
  args: {
    variant: "neutral",
    children: "Neutral Badge",
  },
};

export const Brand: Story = {
  args: {
    variant: "brand",
    children: "Brand Badge",
  },
};

export const Warning: Story = {
  args: {
    variant: "warning",
    children: "Warning Badge",
  },
};

export const Success: Story = {
  args: {
    variant: "success",
    children: "Success Badge",
  },
};

export const Danger: Story = {
  args: {
    variant: "danger",
    children: "Danger Badge",
  },
};

export const Small: Story = {
  args: {
    size: "sm",
    children: "Small Badge",
  },
};

export const Base: Story = {
  args: {
    size: "base",
    children: "Base Badge",
  },
};

export const Large: Story = {
  args: {
    size: "lg",
    children: "Large Badge",
  },
};

export const WithPrependIcon: Story = prependIconExample;

export const WithAppendIcon: Story = appendIconExample;

export const AllVariants: Story = {
  render() {
    return (
      <ExampleGroup title="Primary Variants">
        <Badge variant="neutral">Neutral</Badge>
        <Badge variant="brand">Brand</Badge>
        <Badge variant="warning">Warning</Badge>
        <Badge variant="success">Success</Badge>
        <Badge variant="danger">Danger</Badge>
      </ExampleGroup>
    );
  },
};

export const AllSizes: Story = {
  render() {
    return (
      <ExampleSizes>
        <Badge size="sm">Small</Badge>
        <Badge size="base">Base</Badge>
        <Badge size="lg">Large</Badge>
      </ExampleSizes>
    );
  },
};
