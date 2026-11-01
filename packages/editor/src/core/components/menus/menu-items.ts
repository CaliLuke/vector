/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/react";
import {
  BoldIcon,
  Heading1,
  CheckSquare,
  Heading2,
  Heading3,
  TextQuote,
  ImageIcon,
  TableIcon,
  ListIcon,
  ListOrderedIcon,
  ItalicIcon,
  UnderlineIcon,
  StrikethroughIcon,
  CodeIcon,
  Heading4,
  Heading5,
  Heading6,
  CaseSensitive,
  MinusSquare,
  Palette,
  AlignCenter,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { LinkIcon } from "@plane/propel/icons";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
// helpers
import {
  insertHorizontalRule,
  insertImage,
  insertTableCommand,
  setLinkEditor,
  setText,
  setTextAlign,
  toggleBackgroundColor,
  toggleBlockquote,
  toggleBold,
  toggleBulletList,
  toggleCodeBlock,
  toggleHeading,
  toggleItalic,
  toggleOrderedList,
  toggleStrike,
  toggleTaskList,
  toggleTextColor,
  toggleUnderline,
  unsetLinkEditor,
} from "@/helpers/editor-commands";
// types
import type { TCommandWithProps, TEditorCommands } from "@/types";
import type { ISvgIcons } from "@plane/propel/icons";
type isActiveFunction<T extends TEditorCommands> = (params?: Partial<TCommandWithProps<T>>) => boolean;
type commandFunction<T extends TEditorCommands> = (params?: TCommandWithProps<T>) => void;

export type EditorMenuItem<T extends TEditorCommands> = {
  key: T;
  name: string;
  command: commandFunction<T>;
  icon: LucideIcon | React.FC<ISvgIcons>;
  isActive: isActiveFunction<T>;
};

export const createTextItem = (editor: Editor): EditorMenuItem<"text"> => ({
  key: "text",
  name: "Text",
  isActive: () => editor.isActive(CORE_EXTENSIONS.PARAGRAPH),
  command: () => setText(editor),
  icon: CaseSensitive,
});

type SupportedHeadingLevels = Extract<TEditorCommands, "h1" | "h2" | "h3" | "h4" | "h5" | "h6">;

const createHeadingItem = <T extends SupportedHeadingLevels>(
  editor: Editor,
  level: 1 | 2 | 3 | 4 | 5 | 6,
  key: T,
  name: string,
  icon: LucideIcon
): EditorMenuItem<T> => ({
  key,
  name,
  isActive: () => editor.isActive(CORE_EXTENSIONS.HEADING, { level }),
  command: () => toggleHeading(editor, level),
  icon,
});

export const createHeadingOneItem = (editor: Editor): EditorMenuItem<"h1"> =>
  createHeadingItem(editor, 1, "h1", "Heading 1", Heading1);

export const createHeadingTwoItem = (editor: Editor): EditorMenuItem<"h2"> =>
  createHeadingItem(editor, 2, "h2", "Heading 2", Heading2);

export const createHeadingThreeItem = (editor: Editor): EditorMenuItem<"h3"> =>
  createHeadingItem(editor, 3, "h3", "Heading 3", Heading3);

export const createHeadingFourItem = (editor: Editor): EditorMenuItem<"h4"> =>
  createHeadingItem(editor, 4, "h4", "Heading 4", Heading4);

export const createHeadingFiveItem = (editor: Editor): EditorMenuItem<"h5"> =>
  createHeadingItem(editor, 5, "h5", "Heading 5", Heading5);

export const createHeadingSixItem = (editor: Editor): EditorMenuItem<"h6"> =>
  createHeadingItem(editor, 6, "h6", "Heading 6", Heading6);

export const createBoldItem = (editor: Editor): EditorMenuItem<"bold"> => ({
  key: "bold",
  name: "Bold",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.BOLD),
  command: () => toggleBold(editor),
  icon: BoldIcon,
});

export const createItalicItem = (editor: Editor): EditorMenuItem<"italic"> => ({
  key: "italic",
  name: "Italic",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.ITALIC),
  command: () => toggleItalic(editor),
  icon: ItalicIcon,
});

export const createUnderLineItem = (editor: Editor): EditorMenuItem<"underline"> => ({
  key: "underline",
  name: "Underline",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.UNDERLINE),
  command: () => toggleUnderline(editor),
  icon: UnderlineIcon,
});

export const createStrikeThroughItem = (editor: Editor): EditorMenuItem<"strikethrough"> => ({
  key: "strikethrough",
  name: "Strikethrough",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.STRIKETHROUGH),
  command: () => toggleStrike(editor),
  icon: StrikethroughIcon,
});

export const createBulletListItem = (editor: Editor): EditorMenuItem<"bulleted-list"> => ({
  key: "bulleted-list",
  name: "Bulleted list",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.BULLET_LIST),
  command: () => toggleBulletList(editor),
  icon: ListIcon,
});

export const createNumberedListItem = (editor: Editor): EditorMenuItem<"numbered-list"> => ({
  key: "numbered-list",
  name: "Numbered list",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.ORDERED_LIST),
  command: () => toggleOrderedList(editor),
  icon: ListOrderedIcon,
});

export const createTodoListItem = (editor: Editor): EditorMenuItem<"to-do-list"> => ({
  key: "to-do-list",
  name: "To-do list",
  isActive: () => editor.isActive(CORE_EXTENSIONS.TASK_ITEM),
  command: () => toggleTaskList(editor),
  icon: CheckSquare,
});

export const createQuoteItem = (editor: Editor): EditorMenuItem<"quote"> => ({
  key: "quote",
  name: "Quote",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.BLOCKQUOTE),
  command: () => toggleBlockquote(editor),
  icon: TextQuote,
});

export const createCodeItem = (editor: Editor): EditorMenuItem<"code"> => ({
  key: "code",
  name: "Code",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.CODE_INLINE) || editor?.isActive(CORE_EXTENSIONS.CODE_BLOCK),
  command: () => toggleCodeBlock(editor),
  icon: CodeIcon,
});

const createTableItem = (editor: Editor): EditorMenuItem<"table"> => ({
  key: "table",
  name: "Table",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.TABLE),
  command: () => insertTableCommand(editor),
  icon: TableIcon,
});

const createImageItem = (editor: Editor): EditorMenuItem<"image"> => ({
  key: "image",
  name: "Image",
  isActive: () => editor?.isActive(CORE_EXTENSIONS.IMAGE) || editor?.isActive(CORE_EXTENSIONS.CUSTOM_IMAGE),
  command: () => insertImage({ editor, event: "insert", pos: editor.state.selection.from }),
  icon: ImageIcon,
});

const createHorizontalRuleItem = (editor: Editor): EditorMenuItem<"divider"> =>
  ({
    key: "divider",
    name: "Divider",
    isActive: () => editor?.isActive(CORE_EXTENSIONS.HORIZONTAL_RULE),
    command: () => insertHorizontalRule(editor),
    icon: MinusSquare,
  }) as const;

const createLinkItem = (editor: Editor): EditorMenuItem<"link"> =>
  ({
    key: "link",
    name: "Link",
    isActive: () => editor?.isActive("link"),

    command: (props) => {
      if (!props) return;
      if (props.url) setLinkEditor(editor, props.url, props.text);
      else unsetLinkEditor(editor);
    },

    icon: LinkIcon,
  }) as const;

export const createTextColorItem = (editor: Editor): EditorMenuItem<"text-color"> => ({
  key: "text-color",
  name: "Color",
  isActive: (props) => editor.isActive(CORE_EXTENSIONS.CUSTOM_COLOR, { color: props?.color }),
  command: (props) => {
    if (!props) return;
    toggleTextColor(props.color, editor);
  },
  icon: Palette,
});

export const createBackgroundColorItem = (editor: Editor): EditorMenuItem<"background-color"> => ({
  key: "background-color",
  name: "Background color",
  isActive: (props) => editor.isActive(CORE_EXTENSIONS.CUSTOM_COLOR, { backgroundColor: props?.color }),
  command: (props) => {
    if (!props) return;
    toggleBackgroundColor(props.color, editor);
  },
  icon: Palette,
});

export const createTextAlignItem = (editor: Editor): EditorMenuItem<"text-align"> => ({
  key: "text-align",
  name: "Text align",
  isActive: (props) => editor.isActive({ textAlign: props?.alignment }),
  command: (props) => {
    if (!props) return;
    setTextAlign(props.alignment, editor);
  },
  icon: AlignCenter,
});

export function getEditorMenuItem<T extends TEditorCommands>(
  editor: Editor | null,
  key: T
): EditorMenuItem<T> | undefined {
  if (!editor) return undefined;
  const items: { [K in TEditorCommands]?: EditorMenuItem<K> } = {
    text: createTextItem(editor),
    h1: createHeadingOneItem(editor),
    h2: createHeadingTwoItem(editor),
    h3: createHeadingThreeItem(editor),
    h4: createHeadingFourItem(editor),
    h5: createHeadingFiveItem(editor),
    h6: createHeadingSixItem(editor),
    bold: createBoldItem(editor),
    italic: createItalicItem(editor),
    underline: createUnderLineItem(editor),
    strikethrough: createStrikeThroughItem(editor),
    "bulleted-list": createBulletListItem(editor),
    "to-do-list": createTodoListItem(editor),
    code: createCodeItem(editor),
    "numbered-list": createNumberedListItem(editor),
    quote: createQuoteItem(editor),
    table: createTableItem(editor),
    image: createImageItem(editor),
    divider: createHorizontalRuleItem(editor),
    link: createLinkItem(editor),
    "text-color": createTextColorItem(editor),
    "background-color": createBackgroundColorItem(editor),
    "text-align": createTextAlignItem(editor),
  };
  return items[key];
}
