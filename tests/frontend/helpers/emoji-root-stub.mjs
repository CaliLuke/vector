import { createRequire } from "node:module";

const requireFromWeb = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
const React = requireFromWeb("react");

export function EmojiRoot() {
  return React.createElement("div", { "data-testid": "emoji-root" });
}
