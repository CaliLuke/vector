import { createRequire } from "node:module";

const requireFromWeb = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
const React = requireFromWeb("react");

export function Button({ children, ...props }) {
  return React.createElement("button", props, children);
}
