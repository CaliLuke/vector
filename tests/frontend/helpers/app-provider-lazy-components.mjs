import { React } from "./react-dom.mjs";

export const WEB_SWR_CONFIG = {};
export const TranslationProvider = ({ children }) => React.createElement(React.Fragment, null, children);
export const Toast = () => null;
export const StoreProvider = ({ children }) => React.createElement(React.Fragment, null, children);
export const SWRConfig = ({ children }) => React.createElement(React.Fragment, null, children);
export const resolveGeneralTheme = (theme) => theme;
export const useTheme = () => ({ resolvedTheme: "light" });

export default function LazyProviderComponent({ children }) {
  return children ?? null;
}
