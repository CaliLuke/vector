import { TypeIcon } from "./controls";
import { IconPicker } from "./icon-picker";
import type { PreviewType } from "./preview-content";

type Appearance = Pick<PreviewType, "icon" | "color">;
export function TypeIconPicker({
  value,
  onChange,
  disabled,
}: {
  value: Appearance;
  onChange: (value: Appearance) => void;
  disabled?: boolean;
}) {
  return (
    <IconPicker
      value={value.icon}
      onChange={(icon) => onChange({ ...value, icon })}
      color={{ value: value.color, onChange: (color) => onChange({ ...value, color }) }}
      disabled={disabled}
      label="Choose icon and color"
      trigger={<TypeIcon type={value} />}
    />
  );
}
