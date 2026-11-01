/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { CodeBlockOptions } from "./code-block";
import { CodeBlock } from "./code-block";
import type { Lowlight } from "./lowlight-plugin";
import { LowlightPlugin } from "./lowlight-plugin";

type CodeBlockLowlightOptions = CodeBlockOptions & {
  lowlight: Lowlight | null;
  defaultLanguage: string | null | undefined;
};

export const CodeBlockLowlight = CodeBlock.extend<CodeBlockLowlightOptions>({
  addOptions() {
    return {
      ...(this.parent?.() ?? {
        languageClassPrefix: "language-",
        exitOnTripleEnter: true,
        exitOnArrowDown: true,
        HTMLAttributes: {},
      }),
      lowlight: null,
      defaultLanguage: null,
    };
  },

  addProseMirrorPlugins() {
    return [
      ...(this.parent?.() || []),
      LowlightPlugin({
        name: this.name,
        lowlight: this.options.lowlight,
        defaultLanguage: this.options.defaultLanguage,
      }),
    ];
  },
});
