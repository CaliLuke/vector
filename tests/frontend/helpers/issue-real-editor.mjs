import { existsSync, readFileSync, statSync } from "node:fs";
import { createRequire, registerHooks } from "node:module";
import { transformSync } from "esbuild";
import { fileURLToPath, pathToFileURL } from "node:url";
import { createModuleStub } from "./module-stubs.mjs";

const editorRoot = new URL("../../../packages/editor/src/core/", import.meta.url);
const requireFromEditor = createRequire(new URL("../../../packages/editor/package.json", import.meta.url));
const moduleUrl = (specifier) => {
  const resolved = requireFromEditor.resolve(specifier);
  return pathToFileURL(specifier.startsWith("@tiptap/") ? resolved.replace(/\.cjs$/, ".js") : resolved).href;
};
const helperUrl = new URL("editor-fixture.mjs", import.meta.url).href;

// Keep the production hook, imperative handle, and custom-field adapter. Only
// replace the unrelated editor extensions and workspace upload/mention wrapper.
export function registerIssueRealEditor() {
  registerHooks({
    resolve(specifier, context, nextResolve) {
      if (specifier === "@/components/editor/rich-text" || specifier === helperUrl) {
        return { shortCircuit: true, url: helperUrl };
      }
      if (!context.parentURL?.startsWith(editorRoot.href)) return nextResolve(specifier, context);
      if (specifier === "@/extensions") {
        return createModuleStub(`export { CoreEditorExtensions } from ${JSON.stringify(helperUrl)};`);
      }
      if (specifier === "@/components/menus") return createModuleStub("export const getEditorMenuItems=()=>[];");
      if (specifier === "@/props") return createModuleStub("export const CoreEditorProps=()=>({});");
      if (specifier.startsWith("@/") || specifier.startsWith(".")) {
        const target = specifier.startsWith("@/")
          ? new URL(specifier.slice(2), editorRoot)
          : new URL(specifier, context.parentURL);
        const base = fileURLToPath(target);
        const file = [`${base}.ts`, `${base}.tsx`, `${base}/index.ts`, base].find(
          (candidate) => existsSync(candidate) && statSync(candidate).isFile()
        );
        if (file) return nextResolve(pathToFileURL(file).href, context);
      }
      return nextResolve(specifier, context);
    },
    load(url, context, nextLoad) {
      if (url === helperUrl) {
        return {
          format: "module",
          shortCircuit: true,
          source: `
          import { createElement, forwardRef, useEffect } from ${JSON.stringify(moduleUrl("react"))};
          import { EditorContent } from ${JSON.stringify(moduleUrl("@tiptap/react"))};
          import { Extension } from ${JSON.stringify(moduleUrl("@tiptap/core"))};
          import StarterKit from ${JSON.stringify(moduleUrl("@tiptap/starter-kit"))};
          import { useEditor } from ${JSON.stringify(new URL("hooks/use-editor.ts", editorRoot).href)};
          export const CoreEditorExtensions=()=>[StarterKit.default ?? StarterKit, Extension.create({
            name:"utility", addStorage:()=>({assetsList:[],uploadInProgress:false}),
            addCommands:()=>({
              updateFileHandler:()=>()=>true,
              updateAssetsUploadStatus:()=>()=>true,
              updateIdentityHandlers:(handlers)=>({editor,tr})=>{
                editor.storage.utility.identityHandlers=handlers;
                tr.setMeta("preventDispatch",true);
                return true;
              }
            })
          })];
          const fileHandler={assetsUploadStatus:{}};
          const getEditorMetaData=()=>({file_assets:[],user_mentions:[]});
          export const RichTextEditor=forwardRef((props,ref)=>{
            const fixture=globalThis.issueFormWorkflow;
            fixture.actualEditorRenders=(fixture.actualEditorRenders??0)+1;
            if(fixture.actualEditorRenderLimit && fixture.actualEditorRenders>fixture.actualEditorRenderLimit)
              throw new Error("Editor render loop exceeded regression bound");
            const editor=useEditor({...props,forwardedRef:ref,fileHandler:{...fileHandler},getEditorMetaData,
              enableHistory:true,disabledExtensions:[],flaggedExtensions:[],extendedEditorProps:{}});
            useEffect(()=>{
              if(editor)globalThis.issueFormWorkflow.actualEditors.set(props.id,editor);
            },[editor,props.id]);
            return createElement(EditorContent,{editor});
          });
        `,
        };
      }
      if (url.startsWith(editorRoot.href) && /\.tsx?$/.test(url)) {
        const source = readFileSync(fileURLToPath(url), "utf8");
        return {
          format: "module",
          shortCircuit: true,
          source: transformSync(source, {
            format: "esm",
            loader: url.endsWith(".tsx") ? "tsx" : "ts",
            sourcefile: fileURLToPath(url),
            target: "es2022",
            tsconfigRaw: { compilerOptions: { useDefineForClassFields: true } },
            ...(url.endsWith(".tsx") ? { jsx: "automatic", jsxImportSource: "react" } : {}),
          }).code,
        };
      }
      return nextLoad(url, context);
    },
  });
}

export async function setupRealEditor(t, setup, state) {
  let env;
  t.after(async () => {
    await env?.render(null);
    // TipTap disposes detached views on its next scheduled tick.
    await new Promise((resolve) => setTimeout(resolve, 10));
  });
  env = await setup(t);
  // jsdom has no layout engine; ProseMirror needs Range geometry for focus/undo.
  window.Range.prototype.getClientRects = () => [];
  window.Range.prototype.getBoundingClientRect = () => ({ left: 0, right: 0, top: 0, bottom: 0 });
  state.actualEditors = new Map();
  return env;
}
