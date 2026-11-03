# Bun coverage baseline review

The November 2026 migration replaces V8 coverage with Oxc 0.19.0 on Bun 1.4.4.
Coverage runs instrument workspace source. Separate behavior runs exercise built
package exports, whose source maps can omit executable statement locations.

The review retained all 449 frontend test files and all 1,976 test declarations.
No test titles changed. Upload checks now compare file name, type, and bytes;
the assertion count increased by four. All 449 files passed the fresh Oxc run.
Source patterns and exclusions remain unchanged, covering 2,850 files.

Percentages from the two instrumenters are not interchangeable. Oxc measures
default arguments and statements after early returns independently, and source
instrumentation avoids compiler-generated declarations and lost bundle mappings.
The reviewed baseline conversion raised 1,320 metric floors and lowered 170.
Eight stale entries referred to files already deleted before this migration.
All 42 previously unbaselined files meet the existing 80% new-file requirement.

Version 2 baseline entries contain `[minimumPercentage, maximumUncoveredCount]`.
Both limits apply. New uncovered code fails even when the percentage improves
or the existing percentage is zero. Fixture checks protect both cases and reject
reports without counter data. The 80% requirement for new files is unchanged.

Live, codemods, editor, and UI retain their existing percentage floors. No source
is excluded to make the migration pass. Process-tree memory ceilings and time
budgets are unchanged.
