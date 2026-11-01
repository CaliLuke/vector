/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  AudioIcon,
  CssIcon,
  CsvIcon,
  DefaultIcon,
  DocIcon,
  FigmaIcon,
  HtmlIcon,
  JavaScriptIcon,
  JpgIcon,
  PdfIcon,
  PngIcon,
  RarIcon,
  SheetIcon,
  SvgIcon,
  TxtIcon,
  VideoIcon,
  ZipIcon,
} from "@/components/icons/attachment";

const fileIcons = new Map<string, typeof DefaultIcon>([
  ["pdf", PdfIcon],
  ["csv", CsvIcon],
  ["xlsx", SheetIcon],
  ["css", CssIcon],
  ["doc", DocIcon],
  ["fig", FigmaIcon],
  ["html", HtmlIcon],
  ["png", PngIcon],
  ["jpg", JpgIcon],
  ["js", JavaScriptIcon],
  ["txt", TxtIcon],
  ["svg", SvgIcon],
  ["mp3", AudioIcon],
  ["wav", AudioIcon],
  ["mp4", VideoIcon],
  ["wmv", VideoIcon],
  ["mkv", VideoIcon],
  ["zip", ZipIcon],
  ["rar", RarIcon],
]);

export const getFileIcon = (fileType: string, size: number = 28) => {
  const Icon = fileIcons.get(fileType) ?? DefaultIcon;
  return <Icon height={size} width={size} />;
};
