import { readFileSync } from "node:fs";
import { registerHooks } from "node:module";
import { fileURLToPath } from "node:url";
import { transformFrontendSource } from "./source-transform.mjs";
import { recordCoverageSource } from "./coverage-source.mjs";

const root = new URL("../../../", import.meta.url).href;
registerHooks({
  load(url, context, nextLoad) {
    const owned =
      url.startsWith(root) && /^(apps|packages)\//.test(url.slice(root.length)) && !url.includes("/node_modules/");
    if (!owned || !/\.(?:[cm]?js|tsx?)$/.test(url)) return nextLoad(url, context);
    const filename = fileURLToPath(url);
    if (/\.tsx?$/.test(url)) {
      const output = transformFrontendSource(readFileSync(filename, "utf8"), {
        format: "esm",
        loader: url.endsWith(".tsx") ? "tsx" : "ts",
        sourcefile: filename,
        target: "es2022",
        tsconfigRaw: { compilerOptions: { useDefineForClassFields: true } },
        ...(url.endsWith(".tsx") ? { jsx: "automatic", jsxImportSource: "react" } : {}),
      });
      return { format: "module", shortCircuit: true, source: output };
    }
    const result = nextLoad(url, context);
    if (result.source != null)
      recordCoverageSource(
        filename,
        typeof result.source === "string" ? result.source : Buffer.from(result.source).toString()
      );
    return result;
  },
});
