import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const editorRoot = path.join(repositoryRoot, "packages/editor");
const requireFromEditor = createRequire(path.join(editorRoot, "package.json"));
const typescript = requireFromEditor("typescript");
const warnings = [];
const originalWarn = console.warn;
console.warn = (...args) => warnings.push(args.join(" "));
const candidates = (target) => [
  `${target}.ts`,
  `${target}.tsx`,
  path.join(target, "index.ts"),
  path.join(target, "index.tsx"),
  target,
];

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@/constants/extension") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent('export const CORE_EXTENSIONS = { CUSTOM_LINK: "customLink" };')}`,
      };
    }
    if (specifier === "@/helpers/common") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const isValidHttpUrl = () => ({ isValid: true });")}`,
      };
    }
    if (specifier === "@tiptap/core" || specifier === "@tiptap/starter-kit" || specifier === "linkifyjs") {
      const editorParent = pathToFileURL(path.join(editorRoot, "index.js")).href;
      return nextResolve(specifier, { ...context, parentURL: editorParent });
    }

    let target;
    if (
      (specifier === "." || specifier === ".." || specifier.startsWith("./") || specifier.startsWith("../")) &&
      context.parentURL?.startsWith(pathToFileURL(editorRoot).href)
    ) {
      target = fileURLToPath(new URL(specifier, context.parentURL));
    }

    if (target) {
      const file = candidates(target).find((candidate) => {
        try {
          return requireFromEditor("node:fs").statSync(candidate).isFile();
        } catch {
          return false;
        }
      });
      if (file) return nextResolve(pathToFileURL(file).href, context);
    }

    return nextResolve(specifier, context);
  },
  load(url, context, nextLoad) {
    if (url.startsWith(pathToFileURL(editorRoot).href) && /\.tsx?$/.test(url)) {
      const source = requireFromEditor("node:fs").readFileSync(fileURLToPath(url), "utf8");
      const compilerOptions = {
        module: typescript.ModuleKind.ESNext,
        target: typescript.ScriptTarget.ES2022,
      };
      if (url.endsWith(".tsx")) compilerOptions.jsx = typescript.JsxEmit.ReactJSX;
      return {
        format: "module",
        shortCircuit: true,
        source: typescript.transpileModule(source, { compilerOptions }).outputText,
      };
    }
    return nextLoad(url, context);
  },
});

const [{ Editor }, { default: StarterKit }, { CustomLinkExtension }, { find }] = await Promise.all([
  import("@tiptap/core"),
  import("@tiptap/starter-kit"),
  import("../../packages/editor/src/core/extensions/custom-link/extension.tsx"),
  import("linkifyjs"),
]);

function hasCustomLinkMark(editor, text) {
  return (editor.getJSON().content ?? [])
    .flatMap((node) => node.content ?? [])
    .some(
      (node) =>
        node.text === text && node.marks?.some((mark) => mark.type === "customLink" && mark.attrs?.href === text)
    );
}

test("late custom protocols rebuild linkify and preserve existing editor behavior", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  t.after(() => {
    console.warn = originalWarn;
  });

  const defaultEditor = new Editor({
    element: document.createElement("div"),
    extensions: [StarterKit, CustomLinkExtension],
  });
  let lateCustomEditor;
  let repeatedCustomEditor;
  let postDestroyCustomEditor;

  try {
    defaultEditor.commands.insertContent("https://example.com ");
    assert.equal(hasCustomLinkMark(defaultEditor, "https://example.com"), true);
    assert.equal(
      find("https://example.com").some((link) => link.isLink),
      true
    );

    lateCustomEditor = new Editor({
      element: document.createElement("div"),
      extensions: [
        StarterKit,
        CustomLinkExtension.configure({
          protocols: ["slack"],
          validate: () => true,
        }),
      ],
    });
    lateCustomEditor.commands.insertContent("slack://channel ");
    assert.equal(hasCustomLinkMark(lateCustomEditor, "slack://channel"), true);
    assert.equal(
      find("slack://channel").some((link) => link.isLink),
      true
    );
    defaultEditor.commands.insertContent("https://another.example ");
    assert.equal(hasCustomLinkMark(defaultEditor, "https://another.example"), true);
    assert.equal(
      find("https://another.example").some((link) => link.isLink),
      true
    );

    repeatedCustomEditor = new Editor({
      element: document.createElement("div"),
      extensions: [
        StarterKit,
        CustomLinkExtension.configure({
          protocols: ["slack"],
          validate: () => true,
        }),
      ],
    });
    repeatedCustomEditor.commands.insertContent("slack://another-channel ");
    assert.equal(hasCustomLinkMark(repeatedCustomEditor, "slack://another-channel"), true);
    assert.equal(
      find("slack://another-channel").some((link) => link.isLink),
      true
    );

    lateCustomEditor.destroy();
    lateCustomEditor = undefined;
    repeatedCustomEditor.commands.insertContent("slack://surviving-editor ");
    assert.equal(hasCustomLinkMark(repeatedCustomEditor, "slack://surviving-editor"), true);

    postDestroyCustomEditor = new Editor({
      element: document.createElement("div"),
      extensions: [
        StarterKit,
        CustomLinkExtension.configure({
          protocols: ["slack"],
          validate: () => true,
        }),
      ],
    });
    postDestroyCustomEditor.commands.insertContent("slack://after-destroy ");
    assert.equal(hasCustomLinkMark(postDestroyCustomEditor, "slack://after-destroy"), true);

    assert.equal(warnings.length, 1);
    assert.match(warnings[0], /already initialized.*custom scheme "slack"/);
  } finally {
    defaultEditor.destroy();
    lateCustomEditor?.destroy();
    repeatedCustomEditor?.destroy();
    postDestroyCustomEditor?.destroy();
  }
});
