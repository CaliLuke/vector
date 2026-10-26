/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { parseThemeConfig, type CustomThemeConfig } from "./theme-config";

type Props = { onImport: (value: CustomThemeConfig) => Promise<void>; disabled?: boolean };

export function CustomThemeImportConfigButton({ onImport, disabled }: Props) {
  // refs
  const fileInputRef = useRef<HTMLInputElement>(null);
  // translation
  const { t } = useTranslation();

  const handleUploadConfig = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      await onImport(parseThemeConfig(text));
    } catch (error) {
      console.error("Failed to upload config:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("error"),
        message: error instanceof Error ? error.message : "Failed to import theme configuration",
      });
    } finally {
      // Reset file input
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        disabled={disabled}
        accept=".json"
        onChange={handleUploadConfig}
        className="hidden"
      />
      <Button
        disabled={disabled}
        variant="secondary"
        size="lg"
        type="button"
        onClick={() => fileInputRef.current?.click()}
      >
        Import config
      </Button>
    </>
  );
}
