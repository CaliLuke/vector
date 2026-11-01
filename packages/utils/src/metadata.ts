/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/** Metadata shared by the React Router application entry points. */
export function createAppMetadata(title: string, description: string, keywords: string) {
  return [
    { title },
    { name: "description", content: description },
    { property: "og:title", content: title },
    { property: "og:description", content: description },
    { property: "og:url", content: "https://github.com/CaliLuke/vector" },
    { name: "keywords", content: keywords },
    { name: "twitter:site", content: "Vector" },
  ];
}
