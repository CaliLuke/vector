/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  FileText,
  FileImage,
  FileVideo,
  FileAudio,
  FileArchive,
  FileSpreadsheet,
  FileCode,
  Mail,
  Link2,
} from "lucide-react";
import type { IconType } from "react-icons";
import {
  FaDribbble,
  FaFacebook,
  FaFigma,
  FaGithub,
  FaInstagram,
  FaLinkedin,
  FaTwitter,
  FaYoutube,
  FaChrome,
} from "react-icons/fa6";

type IconMatcher = {
  pattern: RegExp;
  icon: IconType;
};

const SOCIAL_MEDIA_MATCHERS: IconMatcher[] = [
  { pattern: /github\.com/, icon: FaGithub },
  { pattern: /linkedin\.com/, icon: FaLinkedin },
  { pattern: /(twitter\.com|x\.com)/, icon: FaTwitter },
  { pattern: /facebook\.com/, icon: FaFacebook },
  { pattern: /instagram\.com/, icon: FaInstagram },
  { pattern: /youtube\.com/, icon: FaYoutube },
  { pattern: /dribbble\.com/, icon: FaDribbble },
];

const PRODUCTIVITY_MATCHERS: IconMatcher[] = [
  { pattern: /figma\.com/, icon: FaFigma },
  { pattern: /(google\.com|docs\.|doc\.)/, icon: FileText },
];

const FILE_TYPE_MATCHERS: IconMatcher[] = [
  { pattern: /\.(jpg|jpeg|png|gif|bmp|svg|webp)$/, icon: FileImage },
  { pattern: /\.(mp4|mov|avi|wmv|flv|mkv)$/, icon: FileVideo },
  { pattern: /\.(mp3|wav|ogg)$/, icon: FileAudio },
  { pattern: /\.(zip|rar|7z|tar|gz)$/, icon: FileArchive },
  { pattern: /\.(xls|xlsx|csv)$/, icon: FileSpreadsheet },
  { pattern: /\.(pdf|doc|docx|txt)$/, icon: FileText },
  { pattern: /\.(html|js|ts|jsx|tsx|css|scss)$/, icon: FileCode },
];

const OTHER_MATCHERS: IconMatcher[] = [
  { pattern: /^mailto:/, icon: Mail },
  { pattern: /^http/, icon: FaChrome },
];

export const getIconForLink = (url: string) => {
  const lowerUrl = url.toLowerCase();

  const allMatchers = [...SOCIAL_MEDIA_MATCHERS, ...PRODUCTIVITY_MATCHERS, ...FILE_TYPE_MATCHERS, ...OTHER_MATCHERS];

  const matchedIcon = allMatchers.find(({ pattern }) => pattern.test(lowerUrl));
  return matchedIcon?.icon ?? Link2;
};
