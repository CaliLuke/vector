import { createRequire } from "./module-hooks.mjs";

const requireFromWeb = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
const React = requireFromWeb("react");

export function EmojiRoot({ onChange }) {
  return React.createElement(
    "div",
    { "data-testid": "emoji-root" },
    React.createElement(
      "button",
      { type: "button", "data-testid": "choose-emoji", onClick: () => onChange("👍") },
      "Choose thumbs up"
    )
  );
}
