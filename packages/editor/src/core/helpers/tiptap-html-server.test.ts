/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import StarterKitExtension from "@tiptap/starter-kit";
import { generateHTML, generateJSON } from "@tiptap/html/server";

it("round-trips headings, marks, links, and lists through the server HTML API", () => {
  const source =
    '<h2>Release notes</h2><p>Ship <strong>safe</strong> links: <a href="https://example.com/docs">read docs</a></p><ul><li><p>First item</p></li></ul>';

  const content = generateJSON(source, [StarterKitExtension]);
  const html = generateHTML(content, [StarterKitExtension]);

  expect(content.content).toMatchObject([
    { type: "heading", attrs: { level: 2 } },
    {
      type: "paragraph",
      content: [
        {},
        { marks: [{ type: "bold" }], text: "safe" },
        {},
        { marks: [{ type: "link", attrs: { href: "https://example.com/docs" } }], text: "read docs" },
      ],
    },
    { type: "bulletList", content: [{ type: "listItem", content: [{ type: "paragraph" }] }] },
  ]);
  expect(html).toContain("<h2>Release notes</h2>");
  expect(html).toContain("<strong>safe</strong>");
  expect(html).toContain('href="https://example.com/docs"');
  expect(html).toContain("<ul>");
  expect(html).toContain("<li><p>First item</p></li>");
});
