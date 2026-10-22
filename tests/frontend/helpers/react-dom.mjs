import { createRequire } from "node:module";
import { JSDOM } from "jsdom";

const requireFromWeb = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
export const React = requireFromWeb("react");
export const act = React.act;
export const webRequire = requireFromWeb;

const GLOBAL_NAMES = [
  "window",
  "self",
  "document",
  "Document",
  "DocumentFragment",
  "navigator",
  "HTMLElement",
  "Element",
  "Node",
  "Event",
  "MouseEvent",
  "KeyboardEvent",
  "FocusEvent",
  "HTMLButtonElement",
  "HTMLInputElement",
  "HTMLFieldSetElement",
  "HTMLFormElement",
  "HTMLIFrameElement",
  "HTMLLegendElement",
  "HTMLSelectElement",
  "HTMLTextAreaElement",
  "FileList",
  "SVGElement",
  "NodeFilter",
  "MutationObserver",
  "getComputedStyle",
  "requestAnimationFrame",
  "cancelAnimationFrame",
  "IS_REACT_ACT_ENVIRONMENT",
];

export async function createReactDomTestEnvironment() {
  const dom = new JSDOM("<!doctype html><html><body></body></html>", {
    pretendToBeVisual: true,
    url: "http://localhost/",
  });
  const originalGlobals = new Map(
    GLOBAL_NAMES.map((name) => [name, Object.getOwnPropertyDescriptor(globalThis, name)])
  );

  for (const name of GLOBAL_NAMES) {
    const value = name === "IS_REACT_ACT_ENVIRONMENT" ? true : name === "self" ? dom.window : dom.window[name];
    if (value !== undefined) {
      Object.defineProperty(globalThis, name, { configurable: true, writable: true, value });
    }
  }

  const { createRoot } = requireFromWeb("react-dom/client");
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  let disposed = false;

  return {
    container,
    async render(element) {
      await act(async () => root.render(element));
    },
    async unmount() {
      if (disposed) return;
      disposed = true;
      await act(async () => root.unmount());
      dom.window.close();
      for (const [name, descriptor] of originalGlobals) {
        if (descriptor) Object.defineProperty(globalThis, name, descriptor);
        else delete globalThis[name];
      }
    },
  };
}
