# CHANGELOG - @go/db

## [1.0.1] - 2026-05-03
### Optimized
- Refactored `makeResults` to pre-calculate field mappings for structs, significantly improving performance for large result sets.
- Simplified and optimized `makeValue` and `makePublicVarName` functions.
- Optimized time parsing in `makeResults`.

### Fixed
- Fixed typo `isCommitedOrRollbacked` to `isCommittedOrRollbacked` in `Tx` struct.
- Standardized parameter naming: renamed `requestSql` to `query` and `wheres` to `conditions` across the module.
- Modernized Go syntax to align with latest standards.
