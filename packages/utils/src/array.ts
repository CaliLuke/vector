/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import get from "lodash-es/get.js";
import groupByProperty from "lodash-es/groupBy.js";
import type { IIssueLabel, IIssueLabelTree } from "@plane/types";

/**
 * @description Groups an array of objects by a specified key
 * @param {any[]} array Array to group
 * @param {string} key Key to group by (supports dot notation for nested objects)
 * @returns {Object} Grouped object with keys being the grouped values
 * @example
 * const array = [{type: 'A', value: 1}, {type: 'B', value: 2}, {type: 'A', value: 3}];
 * groupBy(array, 'type') // returns { A: [{type: 'A', value: 1}, {type: 'A', value: 3}], B: [{type: 'B', value: 2}] }
 */
export const groupBy = <T>(array: T[], key: string): Record<string, T[]> => {
  const path = key.split(".");
  return groupByProperty(array, (item) => get(item, path) ?? "None");
};

/**
 * @description Orders an array by a specified key in ascending or descending order
 * @param {any[]} orgArray Original array to order
 * @param {string} key Key to order by (supports dot notation for nested objects)
 * @param {"ascending" | "descending"} ordering Sort order
 * @returns {any[]} Ordered array
 * @example
 * const array = [{value: 2}, {value: 1}, {value: 3}];
 * orderArrayBy(array, 'value', 'ascending') // returns [{value: 1}, {value: 2}, {value: 3}]
 */
export const orderArrayBy = <T>(
  orgArray: T[],
  key: string,
  ordering: "ascending" | "descending" = "ascending"
): T[] => {
  if (!orgArray || !Array.isArray(orgArray) || orgArray.length === 0) return [];

  if (key[0] === "-") {
    ordering = "descending";
    key = key.slice(1);
  }

  const innerKey = key.split("."); // split the key by dot

  return orgArray.toSorted((a, b) => {
    const keyA = get(a, innerKey);
    const keyB = get(b, innerKey);
    if (keyA < keyB) {
      return ordering === "ascending" ? -1 : 1;
    }
    if (keyA > keyB) {
      return ordering === "ascending" ? 1 : -1;
    }
    return 0;
  });
};

/**
 * @description Checks if an array contains duplicate values
 * @param {any[]} array Array to check for duplicates
 * @returns {boolean} True if duplicates exist, false otherwise
 * @example
 * checkDuplicates([1, 2, 2, 3]) // returns true
 * checkDuplicates([1, 2, 3]) // returns false
 */
export const checkDuplicates = (array: readonly unknown[]) => new Set(array).size !== array.length;

/**
 * @description Finds the string with the most characters in an array of strings
 * @param {string[]} strings Array of strings to check
 * @returns {string} String with the most characters
 * @example
 * findStringWithMostCharacters(['a', 'bb', 'ccc']) // returns 'ccc'
 */
export const findStringWithMostCharacters = (strings: string[]): string => {
  if (!strings || strings.length === 0) return "";

  return strings.reduce((longestString, currentString) =>
    currentString.length > longestString.length ? currentString : longestString
  );
};

/**
 * @description Checks if two arrays have the same elements regardless of order
 * @param {any[] | null} arr1 First array
 * @param {any[] | null} arr2 Second array
 * @returns {boolean} True if arrays have same elements, false otherwise
 * @example
 * checkIfArraysHaveSameElements([1, 2], [2, 1]) // returns true
 * checkIfArraysHaveSameElements([1, 2], [1, 3]) // returns false
 */
export const checkIfArraysHaveSameElements = (
  arr1: readonly unknown[] | null,
  arr2: readonly unknown[] | null
): boolean => {
  if (!arr1 || !arr2) return false;
  if (!Array.isArray(arr1) || !Array.isArray(arr2)) return false;
  if (arr1.length === 0 && arr2.length === 0) return true;

  return arr1.length === arr2.length && arr1.every((e) => arr2.includes(e));
};

type GroupedItems<T> = { [key: string]: T[] };

/**
 * @description Groups an array of objects by a specified field
 * @param {T[]} array Array to group
 * @param {keyof T} field Field to group by
 * @returns {GroupedItems<T>} Grouped object
 * @example
 * const array = [{type: 'A', value: 1}, {type: 'B', value: 2}];
 * groupByField(array, 'type') // returns { A: [{type: 'A', value: 1}], B: [{type: 'B', value: 2}] }
 */
export const groupByField = <T>(array: T[], field: keyof T): GroupedItems<T> =>
  groupByProperty(array, (item) => String(item[field]));

/**
 * @description Sorts an array of objects by a specified field
 * @param {any[]} array Array to sort
 * @param {string} field Field to sort by
 * @returns {any[]} Sorted array
 * @example
 * const array = [{value: 2}, {value: 1}];
 * sortByField(array, 'value') // returns [{value: 1}, {value: 2}]
 */
export const sortByField = <T>(array: T[], field: keyof T): T[] => {
  array.sort((a, b) => (a[field] < b[field] ? -1 : a[field] > b[field] ? 1 : 0));
  return array;
};

/**
 * @description Orders grouped data by a specified field
 * @param {GroupedItems<T>} groupedData Grouped data object
 * @param {keyof T} orderBy Field to order by
 * @returns {GroupedItems<T>} Ordered grouped data
 */
export const orderGroupedDataByField = <T>(groupedData: GroupedItems<T>, orderBy: keyof T): GroupedItems<T> => {
  for (const key in groupedData) {
    if (Object.hasOwn(groupedData, key)) {
      groupedData[key].sort((a, b) => {
        if (a[orderBy] < b[orderBy]) return -1;
        if (a[orderBy] > b[orderBy]) return 1;
        return 0;
      });
    }
  }
  return groupedData;
};

/**
 * @description Builds a tree structure from an array of labels
 * @param {IIssueLabel[]} array Array of labels
 * @param {any} parent Parent ID
 * @returns {IIssueLabelTree[]} Tree structure
 */
export const buildTree = (array: IIssueLabel[], parent: string | null = null): IIssueLabelTree[] => {
  const tree: IIssueLabelTree[] = [];

  array.forEach((item) => {
    if (item.parent === parent) {
      const children = buildTree(array, item.id);
      tree.push(Object.assign(item, { children }));
    }
  });

  return tree;
};

/**
 * @description Returns valid keys from object whose value is not falsy
 * @param {any} obj Object to check
 * @returns {string[]} Array of valid keys
 * @example
 * getValidKeysFromObject({a: 1, b: 0, c: null}) // returns ['a']
 */
export const getValidKeysFromObject = (obj: unknown): string[] => {
  if (!obj || typeof obj !== "object" || Array.isArray(obj)) return [];

  return Object.entries(obj)
    .filter(([, value]) => Boolean(value))
    .map(([key]) => key);
};

/**
 * @description Converts an array of strings into an object with boolean true values
 * @param {string[]} arrayStrings Array of strings
 * @returns {Object} Object with string keys and boolean values
 * @example
 * convertStringArrayToBooleanObject(['a', 'b']) // returns {a: true, b: true}
 */
export const convertStringArrayToBooleanObject = (arrayStrings: string[]): Record<string, boolean> =>
  Object.fromEntries(arrayStrings.map((value) => [value, true]));

/**
 * @description Sorts dropdown options with selected items appearing first
 * @param {T[]} options Array of dropdown options with value property
 * @param {string[] | string | null | undefined} selectedValues Selected value(s) - array for multi-select, string for single-select
 * @returns {T[]} Sorted array with selected items first
 * @example
 * const options = [{value: '1', label: 'A'}, {value: '2', label: 'B'}];
 * sortBySelectedFirst(options, ['2']) // returns [{value: '2', label: 'B'}, {value: '1', label: 'A'}]
 */
export const sortBySelectedFirst = <T extends { value: string | null }>(
  options: T[] | undefined,
  selectedValues: string[] | string | null | undefined
): T[] | undefined => {
  if (!options || options.length === 0) return options;

  // Normalize selectedValues to array for consistent handling
  const selectedSet = new Set(Array.isArray(selectedValues) ? selectedValues : selectedValues ? [selectedValues] : []);

  if (selectedSet.size === 0) return options;

  // Create a shallow copy to avoid mutating the original array
  return options.toSorted((a, b) => {
    return compareSelection(a.value, b.value, selectedSet);
  });
};

/**
 * @description Sorts dropdown options with current user first, then selected items, then unselected items
 * @param {T[]} options Array of dropdown options with value property
 * @param {string[] | string | null | undefined} selectedValues Selected value(s) - array for multi-select, string for single-select
 * @param {string | undefined} currentUserId ID of the current user to prioritize
 * @returns {T[]} Sorted array with current user first, then selected items, then unselected
 * @example
 * const options = [{value: 'user1'}, {value: 'user2'}, {value: 'user3'}];
 * sortByCurrentUserThenSelected(options, ['user2'], 'user3')
 * // returns [{value: 'user3'}, {value: 'user2'}, {value: 'user1'}]
 */
export const sortByCurrentUserThenSelected = <T extends { value: string | null }>(
  options: T[] | undefined,
  selectedValues: string[] | string | null | undefined,
  currentUserId: string | undefined
): T[] | undefined => {
  if (!options || options.length === 0) return options;

  // Normalize selectedValues to array for consistent handling
  const selectedSet = new Set(Array.isArray(selectedValues) ? selectedValues : selectedValues ? [selectedValues] : []);

  // Create a shallow copy to avoid mutating the original array
  return options.toSorted((a, b) => {
    const aIsCurrent = currentUserId && a.value === currentUserId;
    const bIsCurrent = currentUserId && b.value === currentUserId;

    // Current user always comes first
    if (aIsCurrent && !bIsCurrent) return -1;
    if (!aIsCurrent && bIsCurrent) return 1;
    if (aIsCurrent && bIsCurrent) return 0;

    // If neither is current user, sort by selection state
    return compareSelection(a.value, b.value, selectedSet);
  });
};

function compareSelection(a: string | null, b: string | null, selected: ReadonlySet<string>) {
  const aSelected = a !== null && selected.has(a);
  const bSelected = b !== null && selected.has(b);
  return aSelected === bSelected ? 0 : aSelected ? -1 : 1;
}
