/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// - Parses TipTap/ProseMirror HTML fragments
// - Removes <u> tags (Markdown has no underline)
// - Adds a space after checkbox inputs for correct GFM task list rendering
// - Converts to Markdown using rehype→remark, GFM, and remark-stringify

import type { Element as HASTElement, Parent as HASTParent } from "hast";
import rehypeParse from "rehype-parse";
import rehypeRemark from "rehype-remark";
import remarkGfm from "remark-gfm";
import remarkStringify from "remark-stringify";
import { unified } from "unified";
// local imports
import { parseCustomComponents } from "./custom-components-handler";
import { parseMarks } from "./marks-handler";
import type { TCustomComponentsMetaData } from "./types";

// Rehype plugin to handle TipTap task lists and convert them to GFM-compatible format
// TipTap structure: <li data-type="taskItem"><label><input><span></span></label><div><p>text</p></div></li>
// We need: <li><input> text (with space after checkbox for proper GFM rendering)
function flattenTaskItem(item: HASTElement) {
  const label = item.children.find(
    (child): child is HASTElement => child.type === "element" && child.tagName === "label"
  );
  const content = item.children.find(
    (child): child is HASTElement => child.type === "element" && child.tagName === "div"
  );
  const checkbox = label?.children.find(
    (child): child is HASTElement =>
      child.type === "element" && child.tagName === "input" && child.properties.type === "checkbox"
  );
  if (!checkbox || !content) return;
  const textContent = content.children.flatMap((child) =>
    child.type === "element" && child.tagName === "p" ? child.children : [child]
  );
  item.children = [checkbox, { type: "text", value: " " }, ...textContent];
}

function normalizeTaskLists(node: HASTParent): void {
  for (const child of node.children) {
    if (child.type !== "element") continue;
    if (child.tagName === "li" && child.properties.dataType === "taskItem") flattenTaskItem(child);
    normalizeTaskLists(child);
  }
}

function addSpacesToCheckboxes() {
  return normalizeTaskLists;
}

type TArgs = {
  description_html: string;
  metaData: TCustomComponentsMetaData;
  name?: string;
};

/**
 * Sanitizes a string by escaping HTML entities to prevent XSS attacks
 * @param str - The string to sanitize
 * @returns The sanitized string with escaped HTML entities
 */
function sanitizeHTML(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#x27;");
}

export function convertHTMLToMarkdown(args: TArgs): string {
  const { description_html, metaData, name } = args;

  let updatedDescriptionHtml = description_html;
  if (name) {
    const sanitizedName = sanitizeHTML(name);
    updatedDescriptionHtml = `<h1>${sanitizedName}</h1>\n\n${description_html}`;
  }

  const result = unified()
    .use(rehypeParse, { fragment: true })
    .use(addSpacesToCheckboxes)
    .use(rehypeRemark, {
      handlers: {
        ...parseCustomComponents({
          metaData,
        }),
        ...parseMarks,
      },
    })
    .use(remarkGfm)
    .use(remarkStringify, {
      emphasis: "_",
      handlers: {
        text: (node: { value: string }): string => node.value,
      },
    })
    .processSync(updatedDescriptionHtml);

  const markdown = String(result.value ?? result);

  return markdown;
}
