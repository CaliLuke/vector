import { setTimeout as settleCleanup } from "node:timers/promises";
import { JSDOM } from "jsdom";

import { React, act, webRequire } from "./react.mjs";

const GLOBAL_NAMES = [
  "window",
  "localStorage",
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
  "HTMLTableCellElement",
  "FileList",
  "SVGElement",
  "NodeFilter",
  "MutationObserver",
  "ResizeObserver",
  "getComputedStyle",
  "scrollTo",
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

  // jsdom has no layout engine. Tests can deliver resize notifications explicitly.
  const resizeObservers = new Set();
  class TestResizeObserver {
    targets = new Set();
    constructor(callback) {
      this.callback = callback;
      resizeObservers.add(this);
    }
    observe(target) {
      this.targets.add(target);
    }
    unobserve(target) {
      this.targets.delete(target);
    }
    disconnect() {
      this.targets.clear();
    }
  }
  dom.window.ResizeObserver = TestResizeObserver;
  // jsdom has no viewport to scroll.
  dom.window.scrollTo = () => {};

  for (const name of GLOBAL_NAMES) {
    const value = name === "IS_REACT_ACT_ENVIRONMENT" ? true : name === "self" ? dom.window : dom.window[name];
    if (value !== undefined) {
      Object.defineProperty(globalThis, name, { configurable: true, writable: true, value });
    }
  }

  const { createRoot } = webRequire("react-dom/client");
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  let disposed = false;

  return {
    container,
    async resize(target, contentRect = target.getBoundingClientRect()) {
      await act(async () => {
        for (const observer of resizeObservers) {
          if (observer.targets.has(target)) observer.callback([{ target, contentRect }], observer);
        }
      });
    },
    async render(element) {
      await act(async () => root.render(element));
    },
    async unmount() {
      if (disposed) return;
      disposed = true;
      await act(async () => root.unmount());
      // Allow live-view and source-collection idle cleanup before removing window.
      // The captured real timer also works in tests with mocked application timers.
      await settleCleanup(20);
      resizeObservers.clear();
      dom.window.close();
      for (const [name, descriptor] of originalGlobals) {
        if (descriptor) Object.defineProperty(globalThis, name, descriptor);
        else delete globalThis[name];
      }
    },
  };
}
