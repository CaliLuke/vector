/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { assetDuplicationHandlers } from "@/plane-editor/helpers/asset-duplication";

// Utility function to process HTML content with all registered handlers
export const processAssetDuplication = (htmlContent: string): { processedHtml: string } => {
  const tempDiv = document.createElement("div");
  tempDiv.innerHTML = htmlContent;

  // Legacy private images need the custom node's duplication lifecycle too.
  for (const image of tempDiv.querySelectorAll("img[src]")) {
    const src = image.getAttribute("src") ?? "";
    if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(src)) continue;
    const replacement = document.createElement("image-component");
    for (const attribute of image.attributes) replacement.setAttribute(attribute.name, attribute.value);
    image.replaceWith(replacement);
  }
  // Match serialized tags consistently regardless of the source's quote style.
  let processedHtml = tempDiv.innerHTML;

  // Process each registered component type
  for (const [componentName, handler] of Object.entries(assetDuplicationHandlers)) {
    const elements = tempDiv.querySelectorAll(componentName);

    if (elements.length > 0) {
      elements.forEach((element) => {
        const result = handler({ element, originalHtml: processedHtml });
        if (result.shouldProcess) {
          processedHtml = result.modifiedHtml;
        }
      });

      // Update tempDiv with processed HTML for next iteration
      tempDiv.innerHTML = processedHtml;
    }
  }

  return { processedHtml };
};
