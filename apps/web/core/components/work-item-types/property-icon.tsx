import { createElement } from "react";
import {
  Hash,
  Type,
  Calendar,
  Users,
  ListFilter,
  ToggleLeft,
  Link,
  FileText,
  FunctionSquare,
  GitBranch,
  Package,
} from "lucide-react";
import { typeIconComponents } from "./type-icons";
import type { PropertyKind } from "./preview-content";

const defaultIcons = {
  Text: Type,
  Number: Hash,
  Dropdown: ListFilter,
  Boolean: ToggleLeft,
  Date: Calendar,
  "Member picker": Users,
  "Release picker": Package,
  "Rich text": FileText,
  URL: Link,
  Formula: FunctionSquare,
  Cascading: GitBranch,
};
export function PropertyIcon({ kind, icon }: { kind: PropertyKind; icon?: string }) {
  const Icon = (icon && Object.hasOwn(typeIconComponents, icon) && typeIconComponents[icon]) || defaultIcons[kind];
  return createElement(Icon, { className: "size-4 shrink-0 text-tertiary", "aria-hidden": true });
}
