/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { findChildren } from "@tiptap/core";
import type { Node as ProsemirrorNode } from "@tiptap/pm/model";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { Decoration, DecorationSet } from "@tiptap/pm/view";
import type { createLowlight } from "lowlight";

export type Lowlight = ReturnType<typeof createLowlight>;
type HighlightNode = ReturnType<Lowlight["highlight"]>["children"][number];

function parseNodes(nodes: HighlightNode[], classNames: string[] = []): { text: string; classes: string[] }[] {
  return nodes.flatMap((node) => {
    if (node.type === "element") {
      const classes = Array.isArray(node.properties.className)
        ? node.properties.className.filter((name): name is string => typeof name === "string")
        : [];
      return parseNodes(node.children, [...classNames, ...classes]);
    }
    return node.type === "text" ? [{ text: node.value, classes: classNames }] : [];
  });
}

function getDecorations({
  doc,
  name,
  lowlight,
  defaultLanguage,
}: {
  doc: ProsemirrorNode;
  name: string;
  lowlight: Lowlight;
  defaultLanguage: string | null | undefined;
}) {
  const decorations: Decoration[] = [];

  findChildren(doc, (node) => node.type.name === name).forEach((block) => {
    let from = block.pos + 1;
    const selectedLanguage: unknown = block.node.attrs.language;
    const language = typeof selectedLanguage === "string" && selectedLanguage ? selectedLanguage : defaultLanguage;

    const nodes =
      language && lowlight.registered(language)
        ? lowlight.highlight(language, block.node.textContent).children
        : lowlight.highlightAuto(block.node.textContent).children;

    parseNodes(nodes).forEach((node) => {
      const to = from + node.text.length;

      if (node.classes.length) {
        const decoration = Decoration.inline(from, to, {
          class: node.classes.join(" "),
        });

        decorations.push(decoration);
      }

      from = to;
    });
  });

  return DecorationSet.create(doc, decorations);
}

export function LowlightPlugin({
  name,
  lowlight,
  defaultLanguage,
}: {
  name: string;
  lowlight: Lowlight | null;
  defaultLanguage: string | null | undefined;
}) {
  if (!lowlight) {
    throw Error("You should provide an instance of lowlight to use the code-block-lowlight extension");
  }

  const lowlightPlugin: Plugin<DecorationSet> = new Plugin<DecorationSet>({
    key: new PluginKey("lowlight"),

    state: {
      init: (_, { doc }) =>
        getDecorations({
          doc,
          name,
          lowlight,
          defaultLanguage,
        }),
      apply: (transaction, decorationSet, oldState, newState) => {
        const oldNodeName = oldState.selection.$head.parent.type.name;
        const newNodeName = newState.selection.$head.parent.type.name;
        const oldNodes = findChildren(oldState.doc, (node) => node.type.name === name);
        const newNodes = findChildren(newState.doc, (node) => node.type.name === name);

        if (
          transaction.docChanged &&
          // Apply decorations if:
          // selection includes named node,
          ([oldNodeName, newNodeName].includes(name) ||
            // OR transaction adds/removes named node,
            newNodes.length !== oldNodes.length ||
            // OR transaction has changes that completely encapsulate a node
            // (for example, a transaction that affects the entire document).
            // Such transactions can happen during collab syncing via y-prosemirror, for example.
            transaction.steps.some((step) => {
              if (!("from" in step) || !("to" in step)) return false;
              const { from, to } = step;
              return (
                typeof from === "number" &&
                typeof to === "number" &&
                oldNodes.some((node) => node.pos >= from && node.pos + node.node.nodeSize <= to)
              );
            }))
        ) {
          return getDecorations({
            doc: transaction.doc,
            name,
            lowlight,
            defaultLanguage,
          });
        }

        return decorationSet.map(transaction.mapping, transaction.doc);
      },
    },

    props: {
      decorations(state) {
        return lowlightPlugin.getState(state);
      },
    },
  });

  return lowlightPlugin;
}
