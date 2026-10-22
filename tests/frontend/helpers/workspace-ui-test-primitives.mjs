import { React } from "./react-dom.mjs";

export const EUserPermissions = { ADMIN: "ADMIN", MEMBER: "MEMBER" };
export const ISSUE_DISPLAY_PROPERTIES = [
  { key: "name", titleTranslationKey: "name" },
  { key: "priority", titleTranslationKey: "priority" },
];
export const Menu = {
  Item: function MenuItem({ as = "div", children, ...props }) {
    return React.createElement(as, props, children);
  },
};
export const Settings = () => null;
export const UserPlus = () => null;
export const CheckIcon = () => null;
export const FilterHeader = () => null;
export const observer = (component) => component;
export const useParams = () => ({ workspaceSlug: "default" });
export const useTranslation = () => ({ t: (key) => key });
export const cn = (...values) => {
  const classes = [];
  for (const value of values) {
    if (typeof value === "string") classes.push(value);
    else if (value && typeof value === "object") {
      for (const [className, enabled] of Object.entries(value)) if (enabled) classes.push(className);
    }
  }
  return classes.join(" ");
};
export const getFileURL = (value) => value;
export const getUserRole = (role) => role;

export default function Link({ href, children, ...props }) {
  return React.createElement("a", { href, ...props }, children);
}
