/** Explicit inert operations for filter-presentation tests; scenarios override the behavior they exercise. */
export function filterEditorFixture(overrides) {
  return {
    isConfigReady: true,
    editingError: null,
    hasChanges: false,
    isVisible: true,
    canClear: false,
    canSave: false,
    canUpdate: false,
    onToggleVisibility: () => {},
    onAddCondition: () => {},
    onUpdateConditionProperty: () => {},
    onUpdateConditionOperator: () => {},
    onUpdateConditionValue: () => {},
    onRemoveCondition: () => {},
    onUpdateGroupOperator: () => {},
    onClear: async () => {},
    onSave: async () => {},
    onUpdate: async () => {},
    ...overrides,
  };
}
