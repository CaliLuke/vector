/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ComponentType, ReactNode } from "react";
import { Download, Edit, Share, Star, Archive } from "lucide-react";
import { CopyIcon, TrashIcon } from "../icons";
import { ChevronRightIcon } from "../icons/arrows/chevron-right";
import { ContextMenu } from "./context-menu";

// cannot use satisfies here because base-ui does not have portable types.
const meta: Meta<typeof ContextMenu> = {
  title: "Components/ContextMenu",
  component: ContextMenu,
  subcomponents: {
    ContextMenuTrigger: ContextMenu.Trigger,
    ContextMenuPortal: ContextMenu.Portal,
    ContextMenuContent: ContextMenu.Content,
    ContextMenuItem: ContextMenu.Item,
    ContextMenuSeparator: ContextMenu.Separator,
    ContextMenuSubmenu: ContextMenu.Submenu,
    ContextMenuSubmenuTrigger: ContextMenu.SubmenuTrigger,
  },
  args: {
    children: null,
  },
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

function MenuExample({ trigger, children }: { trigger?: ReactNode; children: ReactNode }) {
  return (
    <ContextMenu>
      <ContextMenu.Trigger>
        {trigger ?? (
          <div className="flex h-[150px] w-[300px] items-center justify-center rounded-md border border-dashed border-strong text-13">
            Right click here
          </div>
        )}
      </ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Content>{children}</ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu>
  );
}

function MenuAction({
  icon: Icon,
  children,
  disabled,
  destructive,
  shortcut,
}: {
  icon: ComponentType<{ className?: string }>;
  children: ReactNode;
  disabled?: boolean;
  destructive?: boolean;
  shortcut?: string;
}) {
  return (
    <ContextMenu.Item disabled={disabled}>
      <Icon className={destructive ? "mr-2 h-4 w-4 text-danger-primary" : "mr-2 h-4 w-4"} />
      {destructive ? <span className="text-danger-primary">{children}</span> : children}
      {shortcut && <span className="ml-auto text-11 text-placeholder">{shortcut}</span>}
    </ContextMenu.Item>
  );
}

function MenuSubmenu({ label, children }: { label: ReactNode; children: ReactNode }) {
  return (
    <ContextMenu.Submenu>
      <ContextMenu.SubmenuTrigger>
        {label}
        <ChevronRightIcon className="ml-auto h-4 w-4" />
      </ContextMenu.SubmenuTrigger>
      <ContextMenu.Portal>
        <ContextMenu.Content>{children}</ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu.Submenu>
  );
}

export const Default: Story = {
  render() {
    return (
      <MenuExample>
        <ContextMenu.Item>Back</ContextMenu.Item>
        <ContextMenu.Item>Forward</ContextMenu.Item>
        <ContextMenu.Item>Reload</ContextMenu.Item>
        <ContextMenu.Separator />
        <ContextMenu.Item>More Tools</ContextMenu.Item>
      </MenuExample>
    );
  },
};

export const WithIcons: Story = {
  render() {
    return (
      <MenuExample>
        <MenuAction icon={CopyIcon}>Copy</MenuAction>
        <MenuAction icon={Edit}>Edit</MenuAction>
        <MenuAction icon={Download}>Download</MenuAction>
        <ContextMenu.Separator />
        <MenuAction icon={Share}>Share</MenuAction>
        <MenuAction icon={TrashIcon} destructive>
          Delete
        </MenuAction>
      </MenuExample>
    );
  },
};

export const WithSubmenus: Story = {
  render() {
    return (
      <MenuExample>
        <MenuAction icon={CopyIcon}>Copy</MenuAction>
        <MenuAction icon={Edit}>Edit</MenuAction>
        <ContextMenu.Separator />
        <MenuSubmenu
          label={
            <>
              <Share className="mr-2 h-4 w-4" />
              Share
            </>
          }
        >
          <ContextMenu.Item>Email</ContextMenu.Item>
          <ContextMenu.Item>Message</ContextMenu.Item>
          <ContextMenu.Item>Copy Link</ContextMenu.Item>
        </MenuSubmenu>
        <ContextMenu.Separator />
        <MenuAction icon={TrashIcon} destructive>
          Delete
        </MenuAction>
      </MenuExample>
    );
  },
};

export const DisabledItems: Story = {
  render() {
    return (
      <MenuExample>
        <MenuAction icon={CopyIcon}>Copy</MenuAction>
        <MenuAction icon={Edit} disabled>
          Edit (Disabled)
        </MenuAction>
        <MenuAction icon={Download}>Download</MenuAction>
        <ContextMenu.Separator />
        <MenuAction icon={Share} disabled>
          Share (Disabled)
        </MenuAction>
        <MenuAction icon={TrashIcon} destructive>
          Delete
        </MenuAction>
      </MenuExample>
    );
  },
};

export const OnFileCard: Story = {
  render() {
    return (
      <MenuExample
        trigger={
          <div className="w-64 cursor-pointer rounded-lg border border-subtle p-4 hover:bg-layer-1">
            <div className="flex items-center gap-3">
              <div className="flex h-12 w-12 items-center justify-center rounded-sm bg-accent-primary text-16 text-on-color">
                📄
              </div>
              <div className="flex-1">
                <div className="font-medium">Document.pdf</div>
                <div className="text-13 text-placeholder">2.4 MB</div>
              </div>
            </div>
          </div>
        }
      >
        <MenuAction icon={Download}>Download</MenuAction>
        <MenuAction icon={CopyIcon}>Copy Link</MenuAction>
        <MenuAction icon={Star}>Add to Favorites</MenuAction>
        <ContextMenu.Separator />
        <MenuAction icon={Archive}>Archive</MenuAction>
        <MenuAction icon={TrashIcon} destructive>
          Delete
        </MenuAction>
      </MenuExample>
    );
  },
};

export const OnImage: Story = {
  render() {
    return (
      <MenuExample
        trigger={
          <div className="relative h-56 w-80 cursor-pointer overflow-hidden rounded-lg bg-layer-1">
            <div className="absolute inset-0 flex items-center justify-center text-placeholder">Image Placeholder</div>
          </div>
        }
      >
        <MenuAction icon={Download}>Save Image</MenuAction>
        <MenuAction icon={CopyIcon}>Copy Image</MenuAction>
        <MenuAction icon={CopyIcon}>Copy Image URL</MenuAction>
        <ContextMenu.Separator />
        <ContextMenu.Item>Open Image in New Tab</ContextMenu.Item>
      </MenuExample>
    );
  },
};

export const OnText: Story = {
  render() {
    return (
      <MenuExample
        trigger={
          <div className="w-96 rounded-lg border border-subtle p-6">
            <h3 className="mb-2 text-16 font-semibold">Context Menu on Text</h3>
            <p className="text-tertiary">
              Right click anywhere on this text area to see the context menu. This demonstrates how context menus can be
              applied to text content areas.
            </p>
          </div>
        }
      >
        <MenuAction icon={CopyIcon}>Copy</MenuAction>
        <MenuAction icon={Edit}>Edit</MenuAction>
        <ContextMenu.Separator />
        <ContextMenu.Item>Select All</ContextMenu.Item>
      </MenuExample>
    );
  },
};

export const NestedSubmenus: Story = {
  render() {
    return (
      <MenuExample>
        <ContextMenu.Item>New File</ContextMenu.Item>
        <ContextMenu.Item>New Folder</ContextMenu.Item>
        <ContextMenu.Separator />
        <MenuSubmenu label="Import">
          <ContextMenu.Item>From File</ContextMenu.Item>
          <ContextMenu.Item>From URL</ContextMenu.Item>
          <MenuSubmenu label="From Cloud">
            <ContextMenu.Item>Google Drive</ContextMenu.Item>
            <ContextMenu.Item>Dropbox</ContextMenu.Item>
            <ContextMenu.Item>OneDrive</ContextMenu.Item>
          </MenuSubmenu>
        </MenuSubmenu>
        <ContextMenu.Separator />
        <MenuAction icon={TrashIcon} destructive>
          Delete
        </MenuAction>
      </MenuExample>
    );
  },
};

export const WithKeyboardShortcuts: Story = {
  render() {
    return (
      <MenuExample>
        <MenuAction icon={CopyIcon} shortcut="⌘C">
          Copy
        </MenuAction>
        <MenuAction icon={Edit} shortcut="⌘E">
          Edit
        </MenuAction>
        <MenuAction icon={Download} shortcut="⌘D">
          Download
        </MenuAction>
        <ContextMenu.Separator />
        <MenuAction icon={TrashIcon} destructive shortcut="⌘⌫">
          Delete
        </MenuAction>
      </MenuExample>
    );
  },
};
