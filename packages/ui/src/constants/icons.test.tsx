/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { LUCIDE_ICONS_LIST as sharedIcons } from "@plane/propel/emoji-icon-picker";
import * as customGlyphs from "@plane/propel/icons";
import { LUCIDE_ICONS_LIST } from "./icons";

describe("persisted icon choices", () => {
  it("retains shared names and ordering without duplicate choices", () => {
    const names = LUCIDE_ICONS_LIST.map(({ name }) => name);
    expect(names).toEqual(sharedIcons.map(({ name }) => name));
    expect(names).toHaveLength(155);
    expect(new Set(names).size).toBe(names.length);
  });

  it.each(
    Object.entries({
      Check: customGlyphs.CheckIcon,
      ChevronDown: customGlyphs.ChevronDownIcon,
      ChevronLeft: customGlyphs.ChevronLeftIcon,
      ChevronRight: customGlyphs.ChevronRightIcon,
      ChevronUp: customGlyphs.ChevronUpIcon,
      Copy: customGlyphs.CopyIcon,
      ExternalLink: customGlyphs.NewTabIcon,
      Globe: customGlyphs.GlobeIcon,
      Link: customGlyphs.LinkIcon,
      Link2: customGlyphs.LinkIcon,
      Lock: customGlyphs.LockIcon,
      Search: customGlyphs.SearchIcon,
    })
  )("keeps the custom glyph for %s", (name, element) => {
    expect(LUCIDE_ICONS_LIST.find((icon) => icon.name === name)?.element).toBe(element);
  });

  it.each(["Github", "Facebook", "Linkedin", "Instagram", "Codepen", "Codesandbox", "Framer", "Map"])(
    "renders the saved %s choice after the icon library upgrade",
    (name) => {
      const icon = LUCIDE_ICONS_LIST.find((entry) => entry.name === name);
      if (!icon) throw new Error(`Missing saved icon: ${name}`);
      const Icon = icon.element;
      const markup = renderToStaticMarkup(<Icon />);
      expect(markup).toContain("<svg");
      expect(markup).toContain("<path");
    }
  );
});
