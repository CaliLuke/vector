import { createRequire } from "./module-hooks.mjs";

const requireFromWeb = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
const React = requireFromWeb("react");

export function Button({ children, ...props }) {
  return React.createElement("button", props, children);
}
