/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import { NODE_FILE_MAP } from "@/plane-editor/constants/utility";

function isFileNodeName(name: string): name is keyof typeof NODE_FILE_MAP {
  return Object.hasOwn(NODE_FILE_MAP, name);
}

/** Document attributes are persisted input; an extension name alone does not validate them. */
export function readFileNode(node: ProseMirrorNode) {
  const name = node.type.name;
  if (!isFileNodeName(name) || typeof node.attrs.src !== "string" || !node.attrs.src) return undefined;
  return {
    node,
    name,
    src: node.attrs.src,
    id: typeof node.attrs.id === "string" ? node.attrs.id : undefined,
    fileSetName: NODE_FILE_MAP[name].fileSetName,
  };
}

export type FileNode = NonNullable<ReturnType<typeof readFileNode>>;

export function addFileSource(sources: Map<string, Set<string>>, file: FileNode) {
  const entries = sources.get(file.name) ?? new Set<string>();
  if (entries.has(file.src)) return false;
  entries.add(file.src);
  sources.set(file.name, entries);
  return true;
}

export function collectFileSources(document: ProseMirrorNode) {
  const sources = new Map<string, Set<string>>();
  document.descendants((node) => {
    const file = readFileNode(node);
    if (!file) return;
    addFileSource(sources, file);
  });
  return sources;
}
