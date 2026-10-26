/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useTranslation } from "@plane/i18n";
import { CloseIcon } from "@plane/propel/icons";
import type { TLogoProps } from "@plane/types";
import { CoverImage } from "@/components/common/cover-image";
import { ImagePickerPopover } from "@/components/core/image-picker-popover";
import { ProjectLogoPicker } from "../project-logo-picker";

type Props = {
  workspaceSlug: string;
  onClose: () => void;
  cover: string | null;
  logo: TLogoProps;
  onCoverChange: (url: string) => void;
  onLogoChange: (value: TLogoProps) => void;
  disabled: boolean;
  busy: boolean;
};
export default function ProjectCreateHeader({
  workspaceSlug,
  onClose,
  cover,
  logo,
  onCoverChange,
  onLogoChange,
  disabled,
  busy,
}: Props) {
  const { t } = useTranslation();
  return (
    <div className="group relative h-44 w-full rounded-lg">
      <CoverImage
        src={cover ?? undefined}
        alt={t("project_cover_image_alt")}
        className="absolute inset-0 size-full rounded-lg"
      />
      <button
        type="button"
        aria-label={t("common.close")}
        disabled={busy}
        onClick={onClose}
        className="absolute top-4 right-4"
      >
        <CloseIcon className="size-5 text-on-color" />
      </button>
      <div className="absolute right-2 bottom-2">
        <ImagePickerPopover
          workspaceSlug={workspaceSlug}
          label={t("change_cover")}
          value={cover}
          onChange={onCoverChange}
          disabled={disabled}
        />
      </div>
      <div className="absolute -bottom-[22px] left-3">
        <ProjectLogoPicker value={logo} onChange={onLogoChange} disabled={disabled} />
      </div>
    </div>
  );
}
