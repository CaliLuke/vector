import { React } from "./react.mjs";

export function ShortcutsModal({ isOpen, onClose }) {
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", { id: "shortcuts-open" }, String(isOpen)),
    React.createElement("button", { id: "close-shortcuts", onClick: onClose })
  );
}
