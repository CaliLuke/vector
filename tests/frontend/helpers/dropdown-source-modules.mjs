import { pathToFileURL } from "node:url";
import { webRequire } from "./react.mjs";
import { createModuleStub } from "./module-stubs.mjs";
import { registerWebTestModules } from "./web-test-modules.mjs";

/** Keep the real dropdown source, React instance and outside-click hook aligned. */
export function registerDropdownSourceModules(dropdownFile, stubs) {
  const reactModules = Object.fromEntries(
    ["react", "react/jsx-runtime", "react/jsx-dev-runtime"].map((name) => [
      name,
      pathToFileURL(webRequire.resolve(name)).href,
    ])
  );
  const outsideClickUrl = new URL("../../../packages/hooks/src/use-outside-click-detector.tsx", import.meta.url).href;
  const platformUrl = new URL("../../../packages/hooks/src/use-platform-os.tsx", import.meta.url).href;
  registerWebTestModules(stubs, (specifier, context, nextResolve) => {
    if (specifier === "@plane/hooks")
      return createModuleStub(
        `export * from ${JSON.stringify(outsideClickUrl)}; export * from ${JSON.stringify(platformUrl)};`
      );
    if (
      specifier === "../utils" &&
      context.parentURL?.includes("/packages/ui/src/dropdowns/") &&
      (dropdownFile === undefined || context.parentURL.endsWith(`/dropdowns/${dropdownFile}`))
    )
      return createModuleStub(
        stubs["@plane/utils/common"] ??
          "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');"
      );
    if (Object.hasOwn(reactModules, specifier)) return { shortCircuit: true, url: reactModules[specifier] };
    return nextResolve(specifier, context);
  });
}
