import { registerHooks } from "node:module";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";

export function registerProjectCardModuleHooks({ React, state, showModalContents }) {
  globalThis.projectCardTestHarness = { ...state, React };
  const element = "const React = globalThis.projectCardTestHarness.React;";
  const empty = "() => null";
  const substitutes = {
    "@plane/hooks": "export const useLocalStorage = () => ({setValue() {}, storedValue:false});",
    "@plane/propel/emoji-icon-picker": `${element} export const Logo = ${empty};`,
    "@plane/propel/icons": `${element} export const LinkIcon = ${empty}, LockIcon = ${empty}, NewTabIcon = ${empty}, TrashIcon = ${empty}, CheckIcon = ${empty};`,
    "@plane/propel/toast":
      'export const setPromiseToast = () => {}; export const setToast = () => {}; export const TOAST_TYPE = {INFO:"info"};',
    "@plane/propel/tooltip": "export const Tooltip = ({children}) => children;",
    "@plane/propel/button": `${element} export const Button = ({children, ...props}) => React.createElement('button', props, children);`,
    "@plane/ui": `${element} export const Avatar = ${empty}, ContextMenu = ${empty};
      export const AvatarGroup = ({children}) => children;
      export const FavoriteStar = ({onClick, selected}) => React.createElement('button', {onClick, 'aria-label':'Favorite project', 'data-selected':selected});`,
    "@plane/utils/common": "export const cn = (...args) => args.filter(x => typeof x === 'string').join(' ');",
    "@plane/utils/file": "export const getFileURL = x => x;",
    "@plane/utils/datetime": "export const renderFormattedDate = x => x;",
    "@plane/utils/string": "export const copyUrlToClipboard = async () => {};",
    "@/hooks/use-platform-os": "export const usePlatformOS = () => ({isMobile:false});",
    "@/components/common/cover-image": `${element} export const CoverImage = ${empty};`,
  };

  for (const [specifier, name] of [
    ["./delete-project-modal", "DeleteProjectModal"],
    ["./join-project-modal", "JoinProjectModal"],
    ["./archive-restore-modal", "ArchiveRestoreProjectModal"],
  ]) {
    const contents = showModalContents
      ? `({isOpen}) => isOpen ? React.createElement('div', {'data-modal':'${name}'}) : null`
      : empty;
    substitutes[specifier] = `${element} export const ${name} = ${contents};`;
  }

  registerWebModuleHooks();
  registerHooks({
    resolve(specifier, context, nextResolve) {
      if (context.parentURL?.endsWith("/project/card.tsx") && Object.hasOwn(substitutes, specifier)) {
        return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(substitutes[specifier])}` };
      }
      return nextResolve(specifier, context);
    },
  });
}
