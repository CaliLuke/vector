/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Plugin, PluginKey } from "@tiptap/pm/state";

export function clickHandler(): Plugin {
  return new Plugin({
    key: new PluginKey("handleClickLink"),
    props: {
      handleClick: (view, _pos, event) => {
        if (event.button !== 0) return false;
        const target = event.target;
        let element = target instanceof Element ? target : target instanceof Node ? target.parentElement : null;
        if (!element || !view.dom.contains(element)) return false;

        // Stay inside the current block and use the clicked anchor, including clicks on its children.
        while (element && element.nodeName !== "DIV") {
          if (element instanceof HTMLAnchorElement) {
            const href = element.href;
            // The browser resolves relative URLs and strips URL whitespace before this protocol check.
            // Keep the blocked schemes aligned with isValidHttpUrl (GHSA-v2vv-7wq3-8w2j).
            if (!href || /^(javascript|data|vbscript|file|about):/i.test(href)) return false;
            window.open(href, element.target);
            return true;
          }
          element = element.parentElement;
        }
        return false;
      },
    },
  });
}
