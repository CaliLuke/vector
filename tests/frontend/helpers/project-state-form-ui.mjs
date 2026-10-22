import { createRequire } from "node:module";
import { Input } from "../../../packages/ui/src/form-fields/input.tsx";

const requireFromWeb = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
const React = requireFromWeb("react");

export { Input };

export function Popover({ button }) {
  return React.createElement("div", null, button);
}

export function TextArea({ value = "", ...props }) {
  return React.createElement("textarea", { ...props, value });
}
