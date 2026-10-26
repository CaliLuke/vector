import { React } from "./react.mjs";

export const TranslationProvider = ({ children }) => React.createElement(React.Fragment, null, children);
export const Toast = () => null;
export const AppStateProvider = ({ children }) => React.createElement(React.Fragment, null, children);
export const resolveGeneralTheme = (theme) => theme;
export const useTheme = () => ({ resolvedTheme: "light" });

export default function LazyProviderComponent({ children }) {
  return children ?? null;
}
