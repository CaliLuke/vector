import assert from "node:assert/strict";
import test from "node:test";
import { act, createReactDomTestEnvironment, React, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { MemoryRouter, useLocation } = webRequire("react-router");

let useRouter;
async function loadUseRouter() {
  if (!useRouter) ({ useRouter } = await import("../../apps/web/app/compat/next/navigation.ts"));
  return useRouter;
}

function CurrentLocation() {
  const { pathname } = useLocation();
  return React.createElement("span", { "data-testid": "pathname" }, pathname);
}

test("a first-render redirect waits for commit and still completes", async () => {
  const environment = await createReactDomTestEnvironment();
  const useRouterHook = await loadUseRouter();
  const warnings = [];
  const originalConsoleWarn = console.warn;
  console.warn = (...args) => warnings.push(args.join(" "));

  function RedirectOnMount() {
    const router = useRouterHook();
    router.push("/destination");
    return React.createElement(CurrentLocation);
  }

  try {
    await environment.render(
      React.createElement(
        React.StrictMode,
        null,
        React.createElement(MemoryRouter, { initialEntries: ["/start"] }, React.createElement(RedirectOnMount))
      )
    );

    await act(async () => new Promise((resolve) => setTimeout(resolve, 10)));
    assert.equal(
      environment.container.querySelector("[data-testid='pathname']")?.textContent,
      "/destination/",
      warnings.join("\n")
    );
    assert.equal(
      warnings.some((warning) => warning.includes("call navigate() in a React.useEffect()")),
      false
    );
  } finally {
    console.warn = originalConsoleWarn;
    await environment.unmount();
  }
});

test("a navigation requested by an abandoned suspended render is discarded", async () => {
  const environment = await createReactDomTestEnvironment();
  const useRouterHook = await loadUseRouter();
  const warnings = [];
  const originalConsoleWarn = console.warn;
  console.warn = (...args) => warnings.push(args.join(" "));
  const unresolved = new Promise(() => {});

  function SuspendedRedirect() {
    const router = useRouterHook();
    router.push("/stale");
    throw unresolved;
  }

  try {
    await environment.render(
      React.createElement(
        MemoryRouter,
        { initialEntries: ["/start"] },
        React.createElement(CurrentLocation),
        React.createElement(
          React.Suspense,
          { fallback: React.createElement("span", null, "Loading") },
          React.createElement(SuspendedRedirect)
        )
      )
    );
    await new Promise((resolve) => setTimeout(resolve, 10));

    assert.equal(environment.container.querySelector("[data-testid='pathname']")?.textContent, "/start");
    assert.equal(
      warnings.some((warning) => warning.includes("call navigate() in a React.useEffect()")),
      false
    );
  } finally {
    console.warn = originalConsoleWarn;
    await environment.unmount();
  }
});

test("a committed click navigates even when its modal closes immediately", async () => {
  const environment = await createReactDomTestEnvironment();
  const useRouterHook = await loadUseRouter();

  function NavigationButton({ onNavigate }) {
    const router = useRouterHook();
    return React.createElement(
      "button",
      {
        onClick: () => {
          router.push("/created-item");
          onNavigate();
        },
      },
      "Navigate"
    );
  }

  function Harness() {
    const [showButton, setShowButton] = React.useState(true);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(CurrentLocation),
      showButton ? React.createElement(NavigationButton, { onNavigate: () => setShowButton(false) }) : null
    );
  }

  try {
    await environment.render(
      React.createElement(MemoryRouter, { initialEntries: ["/start"] }, React.createElement(Harness))
    );
    await act(async () => environment.container.querySelector("button").click());
    await act(async () => new Promise((resolve) => setTimeout(resolve, 10)));

    assert.equal(environment.container.querySelector("[data-testid='pathname']")?.textContent, "/created-item/");
    assert.equal(environment.container.querySelector("button"), null);
  } finally {
    await environment.unmount();
  }
});
