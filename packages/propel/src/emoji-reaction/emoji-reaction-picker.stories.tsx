/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useState } from "react";
import { useExampleReactions } from "./reaction-examples";
import { ReactionMessageExample } from "./message-example";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { SmilePlus } from "lucide-react";
import { stringToEmoji } from "../emoji-icon-picker";
import { EmojiReactionGroup } from "./emoji-reaction";
import type { EmojiReactionPickerProps } from "./emoji-reaction-picker";
import { EmojiReactionPicker } from "./emoji-reaction-picker";

const meta = {
  title: "Components/Emoji/EmojiReactionPicker",
  component: EmojiReactionPicker,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof EmojiReactionPicker>;

export default meta;
type Story = StoryObj<typeof meta>;

function SelectedEmojiExample({
  renderLabel,
  resultClassName = "text-13",
  ...pickerProps
}: Pick<EmojiReactionPickerProps, "searchDisabled" | "searchPlaceholder"> & {
  renderLabel: (selected: string | null) => ReactNode;
  resultClassName?: string;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedEmoji, setSelectedEmoji] = useState<string | null>(null);
  return (
    <div className="space-y-4 p-4">
      <EmojiReactionPicker
        {...pickerProps}
        isOpen={isOpen}
        handleToggle={setIsOpen}
        onChange={setSelectedEmoji}
        closeOnSelect
        label={renderLabel(selectedEmoji)}
      />
      {selectedEmoji && <div className={resultClassName}>Selected: {selectedEmoji}</div>}
    </div>
  );
}

export const Default: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "Pick Emoji",
  },
  render() {
    return (
      <SelectedEmojiExample
        resultClassName="rounded-sm border border-subtle bg-layer-1 p-4 text-13"
        renderLabel={(selectedEmoji) => (
          <span className="flex size-8 items-center justify-center rounded-md px-2 text-18">
            {selectedEmoji ? stringToEmoji(selectedEmoji) : <SmilePlus className="h-6 text-primary" />}
          </span>
        )}
      />
    );
  },
};

export const WithCustomLabel: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "Add Reaction",
  },
  render() {
    return (
      <SelectedEmojiExample
        renderLabel={(selectedEmoji) => (
          <button className="flex items-center gap-2 rounded-sm border border-subtle bg-layer-1 px-4 py-2 hover:bg-surface-2">
            {selectedEmoji ? stringToEmoji(selectedEmoji) : <SmilePlus className="h-4 w-4" />}
            <span className="text-13">Add Reaction</span>
          </button>
        )}
      />
    );
  },
};

export const InlineReactions: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "Add",
  },
  render() {
    const [isOpen, setIsOpen] = useState(false);
    const {
      reactions,
      addReaction: handleReactionAdd,
      toggleReaction: handleReactionClick,
    } = useExampleReactions([
      { emoji: "👍", count: 3, reacted: false, users: ["Alice", "Bob", "Charlie"] },
      { emoji: "❤️", count: 2, reacted: true, users: ["You", "David"] },
    ]);

    return (
      <div className="p-4">
        <EmojiReactionGroup
          reactions={reactions}
          onReactionClick={handleReactionClick}
          onAddReaction={() => setIsOpen(true)}
          showAddButton={false}
        />
        <EmojiReactionPicker
          isOpen={isOpen}
          handleToggle={setIsOpen}
          onChange={handleReactionAdd}
          closeOnSelect
          label={
            <button className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-dashed border-strong bg-surface-1 text-placeholder transition-all duration-200 hover:border-accent-strong hover:bg-accent-primary/5 hover:text-accent-primary">
              <SmilePlus className="h-3.5 w-3.5" />
            </button>
          }
        />
      </div>
    );
  },
};

export const DifferentPlacements: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "Placements",
  },
  render() {
    const [isOpen1, setIsOpen1] = useState(false);
    const [isOpen2, setIsOpen2] = useState(false);
    const [isOpen3, setIsOpen3] = useState(false);
    const [isOpen4, setIsOpen4] = useState(false);

    return (
      <div className="space-y-8 p-8">
        <div className="flex items-center gap-4">
          <span className="w-32 text-13">Bottom Start:</span>
          <EmojiReactionPicker
            isOpen={isOpen1}
            handleToggle={setIsOpen1}
            onChange={() => {}}
            placement="bottom-start"
            label={<SmilePlus className="h-6 w-6" />}
          />
        </div>
        <div className="flex items-center gap-4">
          <span className="w-32 text-13">Bottom End:</span>
          <EmojiReactionPicker
            isOpen={isOpen2}
            handleToggle={setIsOpen2}
            onChange={() => {}}
            placement="bottom-end"
            label={<SmilePlus className="h-6 w-6" />}
          />
        </div>
        <div className="flex items-center gap-4">
          <span className="w-32 text-13">Top Start:</span>
          <EmojiReactionPicker
            isOpen={isOpen3}
            handleToggle={setIsOpen3}
            onChange={() => {}}
            placement="top-start"
            label={<SmilePlus className="h-6 w-6" />}
          />
        </div>
        <div className="flex items-center gap-4">
          <span className="w-32 text-13">Top End:</span>
          <EmojiReactionPicker
            isOpen={isOpen4}
            handleToggle={setIsOpen4}
            onChange={() => {}}
            placement="top-end"
            label={<SmilePlus className="h-6 w-6" />}
          />
        </div>
      </div>
    );
  },
};

export const SearchDisabled: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "No Search",
  },
  render() {
    return (
      <SelectedEmojiExample
        searchDisabled
        renderLabel={() => (
          <button className="rounded-sm border border-subtle bg-layer-1 px-4 py-2 hover:bg-surface-2">No Search</button>
        )}
      />
    );
  },
};

export const CustomSearchPlaceholder: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "Custom Search",
  },
  render() {
    return (
      <SelectedEmojiExample
        searchPlaceholder="Find your emoji..."
        renderLabel={() => (
          <button className="rounded-sm border border-subtle bg-layer-1 px-4 py-2 hover:bg-surface-2">
            Custom Search
          </button>
        )}
      />
    );
  },
};

export const CloseOnSelectDisabled: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "Select Multiple",
  },
  render() {
    const [isOpen, setIsOpen] = useState(false);
    const [selectedEmojis, setSelectedEmojis] = useState<{ id: string; emoji: string }[]>([]);

    const handleChange = (emoji: string) => {
      const selection = { id: crypto.randomUUID(), emoji };
      setSelectedEmojis((prev) => [...prev, selection]);
    };

    return (
      <div className="space-y-4 p-4">
        <div className="flex items-center gap-2">
          <EmojiReactionPicker
            isOpen={isOpen}
            handleToggle={setIsOpen}
            onChange={handleChange}
            closeOnSelect={false}
            label={
              <button className="rounded-sm border border-subtle bg-layer-1 px-4 py-2 hover:bg-surface-2">
                Select Multiple (Stays Open)
              </button>
            }
          />
          <button
            className="rounded-sm bg-layer-1 px-3 py-1.5 text-13 hover:bg-surface-2"
            onClick={() => setSelectedEmojis([])}
          >
            Clear
          </button>
        </div>
        {selectedEmojis.length > 0 && (
          <div className="rounded-sm border border-subtle bg-layer-1 p-4 text-13">
            <div className="mb-2 font-medium">Selected ({selectedEmojis.length}):</div>
            <div className="flex flex-wrap gap-2">
              {selectedEmojis.map(({ emoji, id }) => (
                <span key={id} className="text-18">
                  {emoji}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    );
  },
};

export const InMessageContext: Story = {
  args: {
    isOpen: false,
    handleToggle: () => {},
    onChange: () => {},
    label: "Message",
  },
  render() {
    const [isOpen, setIsOpen] = useState(false);
    const {
      reactions,
      addReaction: handleReactionAdd,
      toggleReaction: handleReactionClick,
    } = useExampleReactions([{ emoji: "👍", count: 5, reacted: false, users: ["Alice", "Bob", "Charlie"] }]);

    return (
      <ReactionMessageExample message="Just finished the design for the new dashboard! Would love to hear your thoughts.">
        <div className="flex gap-2">
          <EmojiReactionGroup reactions={reactions} onReactionClick={handleReactionClick} showAddButton={false} />
          <EmojiReactionPicker
            isOpen={isOpen}
            handleToggle={setIsOpen}
            onChange={handleReactionAdd}
            closeOnSelect
            label={
              <button className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-dashed border-strong bg-surface-1 text-placeholder transition-all duration-200 hover:border-accent-strong hover:bg-accent-primary/5 hover:text-accent-primary">
                <SmilePlus className="h-3.5 w-3.5" />
              </button>
            }
          />
        </div>
      </ReactionMessageExample>
    );
  },
};
