import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { findSidebarScrollViewport } from "../../apps/web/core/components/workspace/sidebar/projects-list.utils.ts";

registerWebModuleHooks();
const sidebarPrimitives = new URL("./helpers/sidebar-test-primitives.mjs", import.meta.url).href;
const dragHandleSource = new URL("../../packages/ui/src/drag-handle.tsx", import.meta.url).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/ui") {
      return { shortCircuit: true, url: dragHandleSource };
    }
    if (specifier === "@plane/utils") {
      return { shortCircuit: true, url: sidebarPrimitives };
    }
    if (specifier === "./utils" && context.parentURL?.endsWith("/packages/ui/src/drag-handle.tsx")) {
      return { shortCircuit: true, url: sidebarPrimitives };
    }
    return nextResolve(specifier, context);
  },
});
const { SidebarDragHandle } = await import("../../apps/web/core/components/workspace/sidebar/drag-handle.tsx");

test("project list auto-scroll resolves to its nearest scroll-area viewport", async () => {
  const environment = await createReactDomTestEnvironment();

  try {
    const outerViewport = document.createElement("div");
    outerViewport.dataset.slot = "scroll-area-viewport";
    outerViewport.style.overflowY = "auto";
    const innerViewport = document.createElement("div");
    innerViewport.dataset.slot = "scroll-area-viewport";
    innerViewport.style.overflowY = "auto";
    const listContainer = document.createElement("div");
    outerViewport.append(innerViewport);
    innerViewport.append(listContainer);
    environment.container.append(outerViewport);

    assert.equal(findSidebarScrollViewport(listContainer), innerViewport);
    assert.equal(findSidebarScrollViewport(environment.container), null);
  } finally {
    await environment.unmount();
  }
});

test("sidebar drag handle tooltip keeps one button and forwards its ref to it", async () => {
  const environment = await createReactDomTestEnvironment();
  const dragHandleRef = React.createRef();

  try {
    await environment.render(
      React.createElement(SidebarDragHandle, {
        handleClassName: "sidebar-project-drag-handle",
        position: "top-start",
        tooltipContent: "Drag to rearrange",
        ref: dragHandleRef,
      })
    );

    const dragHandle = environment.container.querySelector("button.sidebar-project-drag-handle");
    assert.ok(dragHandle);
    assert.equal(environment.container.querySelectorAll("button").length, 1);
    assert.equal(dragHandle.querySelector("button"), null);
    assert.equal(dragHandleRef.current, dragHandle);
  } finally {
    await environment.unmount();
  }
});
