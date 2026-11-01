/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import groupBy from "lodash-es/groupBy.js";

// plane imports
import { RANDOM_EMOJI_CODES } from "@plane/constants";

/**
 * Converts a hyphen-separated hexadecimal emoji code to its decimal representation
 * @param {string} emojiUnified - The unified emoji code in hexadecimal format (e.g., "1f600" or "1f1e6-1f1e8")
 * @returns {string} The decimal representation of the emoji code (e.g., "128512" or "127462-127464")
 * @example
 * convertHexEmojiToDecimal("1f600") // returns "128512"
 * convertHexEmojiToDecimal("1f1e6-1f1e8") // returns "127462-127464"
 * convertHexEmojiToDecimal("") // returns ""
 */
export const convertHexEmojiToDecimal = (emojiUnified: string): string => {
  if (!emojiUnified) return "";

  return emojiUnified
    .split("-")
    .map((e) => parseInt(e, 16))
    .join("-");
};

/**
 * Converts a hyphen-separated decimal emoji code back to its hexadecimal representation
 * @param {string} emoji - The emoji code in decimal format (e.g., "128512" or "127462-127464")
 * @returns {string} The hexadecimal representation of the emoji code (e.g., "1f600" or "1f1e6-1f1e8")
 * @example
 * emojiCodeToUnicode("128512") // returns "1f600"
 * emojiCodeToUnicode("127462-127464") // returns "1f1e6-1f1e8"
 * emojiCodeToUnicode("") // returns ""
 */
export const emojiCodeToUnicode = (emoji: string): string => {
  if (!emoji) return "";

  // convert emoji code to unicode
  const uniCodeEmoji = emoji
    .split("-")
    .map((codePoint) => parseInt(codePoint, 10).toString(16))
    .join("-");

  return uniCodeEmoji;
};

/**
 * Groups reactions by a specified key
 * @param reactions - Array of reaction objects
 * @param {string} key - Key to group reactions by
 * @returns {Object} Object with reactions grouped by the specified key
 */
export const groupReactions = <T extends object>(reactions: T[], key: keyof T): Record<string, T[]> => {
  if (!Array.isArray(reactions)) {
    console.error("Expected an array of reactions, but got:", reactions);
    return {};
  }

  const validReactions = reactions.filter((reaction) => {
    if (!reaction || typeof reaction !== "object" || !Object.hasOwn(reaction, key)) {
      console.warn("Skipping undefined reaction or missing key:", reaction);
      return false;
    }
    return true;
  });
  return groupBy(validReactions, (reaction) => reaction[key]);
};

/**
 * Returns a random emoji code from the RANDOM_EMOJI_CODES array
 * @returns {string} A random emoji code
 */
export const getRandomEmoji = (): string => RANDOM_EMOJI_CODES[Math.floor(Math.random() * RANDOM_EMOJI_CODES.length)];
