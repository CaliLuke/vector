import type { IGroupByColumn, TIssuesResponse } from "@plane/types";

export function parentGroupColumns(parents: NonNullable<TIssuesResponse["parent_groups"]>): IGroupByColumn[] {
  return [
    ...parents.map((parent) => ({
      id: parent.id,
      name: `${parent.identifier} ${parent.name}`,
      payload: { parent_id: parent.id },
      isDropDisabled: true,
    })),
    { id: "None", name: "No matching parent", payload: {}, isDropDisabled: true },
  ];
}
