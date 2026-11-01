/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mergeAttributes, Node, textblockTypeInputRule } from "@tiptap/core";
import type { NodeType } from "@tiptap/pm/model";
import type { EditorView } from "@tiptap/pm/view";
import { Plugin, PluginKey } from "@tiptap/pm/state";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";

export type CodeBlockOptions = {
  /**
   * Adds a prefix to language classes that are applied to code tags.
   * Defaults to `'language-'`.
   */
  languageClassPrefix: string;
  /**
   * Define whether the node should be exited on triple enter.
   * Defaults to `true`.
   */
  exitOnTripleEnter: boolean;
  /**
   * Define whether the node should be exited on arrow down if there is no node after it.
   * Defaults to `true`.
   */
  exitOnArrowDown: boolean;
  /**
   * Custom HTML attributes that should be added to the rendered HTML tag.
   */
  HTMLAttributes: Record<string, unknown>;
};

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    [CORE_EXTENSIONS.CODE_BLOCK]: {
      /**
       * Set a code block
       */
      setCodeBlock: (attributes?: { language: string }) => ReturnType;
      /**
       * Toggle a code block
       */
      toggleCodeBlock: (attributes?: { language: string }) => ReturnType;
    };
  }
}

const backtickInputRegex = /^```([a-z]+)?[\s\n]$/;
const tildeInputRegex = /^~~~([a-z]+)?[\s\n]$/;

function pasteInlineCode(view: EditorView, text: string): boolean {
  const { tr, schema } = view.state;
  const { from, to } = tr.selection;
  const code = schema.marks.code;
  if (!code || from > to || from < 0 || to > tr.doc.content.size) return false;
  tr.replaceWith(from, to, schema.text(text, [code.create()]));
  view.dispatch(tr);
  return true;
}

function pasteCodeBlock(view: EditorView, type: NodeType, text: string, metadata: string): boolean {
  if (!metadata) return false;
  const data: unknown = JSON.parse(metadata);
  if (!data || typeof data !== "object" || !("mode" in data) || typeof data.mode !== "string" || !data.mode)
    return false;
  const { tr, schema } = view.state;
  const { $from } = tr.selection;
  const insertPos = !$from.parent.textContent.trim() ? $from.pos - 1 : $from.end($from.depth) + 1;
  if (insertPos < 0 || insertPos > tr.doc.content.size) return false;
  const textNode = schema.text(text.replace(/\r\n?/g, "\n"));
  tr.insert(insertPos, type.create({ language: data.mode }, textNode));
  view.dispatch(tr);
  return true;
}

export const CodeBlock = Node.create<CodeBlockOptions>({
  name: CORE_EXTENSIONS.CODE_BLOCK,

  addOptions() {
    return {
      languageClassPrefix: "language-",
      exitOnTripleEnter: true,
      exitOnArrowDown: true,
      HTMLAttributes: {},
    };
  },
  content: "text*",

  marks: "",

  group: "block",

  code: true,

  defining: true,

  addAttributes() {
    return {
      language: {
        default: null,
        parseHTML: (element) => {
          const { languageClassPrefix } = this.options;
          const classNames = [...(element.firstElementChild?.classList || [])];
          const languages = classNames
            .filter((className) => className.startsWith(languageClassPrefix))
            .map((className) => className.replace(languageClassPrefix, ""));
          const language = languages[0];

          if (!language) {
            return null;
          }

          return language;
        },
        rendered: false,
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: "pre",
        preserveWhitespace: "full",
      },
    ];
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      "pre",
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes),
      [
        "code",
        {
          class: node.attrs.language ? this.options.languageClassPrefix + node.attrs.language : null,
        },
        0,
      ],
    ];
  },

  addCommands() {
    return {
      setCodeBlock:
        (attributes) =>
        ({ commands }) =>
          commands.setNode(this.name, attributes),
      toggleCodeBlock:
        (attributes) =>
        ({ commands }) =>
          commands.toggleNode(this.name, CORE_EXTENSIONS.PARAGRAPH, attributes),
    };
  },

  addKeyboardShortcuts() {
    return {
      "Mod-Alt-c": () => this.editor.commands.toggleCodeBlock(),

      // remove codeBlock when at start of document or codeBlock is empty
      Backspace: () => {
        try {
          const { empty, $anchor } = this.editor.state.selection;
          const isAtStart = $anchor.pos === 1;

          if (!empty || $anchor.parent.type.name !== this.name) {
            return false;
          }

          if (isAtStart || !$anchor.parent.textContent.length) {
            return this.editor.commands.clearNodes();
          }

          return false;
        } catch (error) {
          console.error("Error handling Backspace in code block:", error);
          return false;
        }
      },

      // exit node on triple enter
      Enter: ({ editor }) => {
        try {
          if (!this.options.exitOnTripleEnter) {
            return false;
          }

          const { state } = editor;
          const { selection } = state;
          const { $from, empty } = selection;

          if (!empty || $from.parent.type !== this.type) {
            return false;
          }

          const isAtEnd = $from.parentOffset === $from.parent.nodeSize - 2;
          const endsWithDoubleNewline = $from.parent.textContent.endsWith("\n\n");

          if (!isAtEnd || !endsWithDoubleNewline) {
            return false;
          }

          return editor
            .chain()
            .command(({ tr }) => {
              tr.delete($from.pos - 2, $from.pos);

              return true;
            })
            .exitCode()
            .run();
        } catch (error) {
          console.error("Error handling Enter in code block:", error);
          return false;
        }
      },

      // exit node on arrow down
      ArrowDown: ({ editor }) => {
        try {
          if (!this.options.exitOnArrowDown) {
            return false;
          }

          const { state } = editor;
          const { selection, doc } = state;
          const { $from, empty } = selection;

          if (!empty || $from.parent.type !== this.type) {
            return false;
          }

          const isAtEnd = $from.parentOffset === $from.parent.nodeSize - 2;

          if (!isAtEnd) {
            return false;
          }

          const after = $from.after();

          if (after === undefined) {
            return false;
          }

          const nodeAfter = doc.nodeAt(after);

          if (nodeAfter) {
            return false;
          }

          return editor.commands.exitCode();
        } catch (error) {
          console.error("Error handling ArrowDown in code block:", error);
          return false;
        }
      },
    };
  },

  addInputRules() {
    return [
      textblockTypeInputRule({
        find: backtickInputRegex,
        type: this.type,
        getAttributes: (match) => ({
          language: match[1],
        }),
      }),
      textblockTypeInputRule({
        find: tildeInputRegex,
        type: this.type,
        getAttributes: (match) => ({
          language: match[1],
        }),
      }),
    ];
  },
  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey("codeBlockVSCodeHandlerCustom"),
        props: {
          handlePaste: (view, event) => {
            try {
              if (!event.clipboardData || this.editor.isActive(this.type.name)) return false;
              const text = event.clipboardData.getData("text/plain");
              if (!text) return false;
              const handled = this.editor.isActive(CORE_EXTENSIONS.CODE_INLINE)
                ? pasteInlineCode(view, text)
                : pasteCodeBlock(view, this.type, text, event.clipboardData.getData("vscode-editor-data"));
              if (handled) event.preventDefault();
              return handled;
            } catch (error) {
              console.error("Error handling paste in CodeBlock extension:", error);
              return false;
            }
          },
        },
      }),
    ];
  },
});
