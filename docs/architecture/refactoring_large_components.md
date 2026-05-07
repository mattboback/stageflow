# Comprehensive Frontend Refactoring Plan: Large Components

## 🎯 Objectives & Rationale
Currently, three critical frontend components have grown excessively large (800 - 1200+ lines): `EditorShell`, `TemplateCustomizer`, and `PromptEditorCard`. These files mix complex state management, data parsing, pure layout, and sub-components.

By extracting these down into collocated, focused files, we will achieve:
1. **Better Readability**: Orchestration components will become lightweight entry points (~150-200 lines) representing the high-level layout.
2. **Improved Testability**: Isolated state hooks, sub-components, and pure functions can be unit-tested without mounting heavy parent contexts.
3. **Strict Type Boundaries**: Presentational components will use explicitly defined props rather than being coupled to complex page-state hook return types.
4. **Reduced Merge Conflicts**: High-churn domains are spread across smaller, isolated files.

---

## 🏗️ Architectural Changes

### 1. EditorShell
**Target:** Shrink `views/EditorShell.tsx` from 1231 lines to ~200 lines.
**Strategy:** Extract the desktop/mobile structural divergence, separate the preview logic, and enforce explicit type contracts.

```text
src/features/applications/detail/views/
├── EditorShell.tsx (Main orchestrator, ~200 lines)
└── EditorShell/
    ├── DesktopEditorWorkspace.tsx
    ├── MobileEditorWorkspace.tsx
    ├── DesktopPreviewWorkspace.tsx
    ├── MobilePreviewWorkspace.tsx
    ├── PreviewDocumentFrame.tsx
    ├── GenerationFailureView.tsx
    ├── ApplicationEditorDialogs.tsx
    ├── types.ts (Explicit prop types)
    └── utils.ts (isMobileTabValue, normalizePreviewText, etc.)
```

### 2. TemplateCustomizer
**Target:** Shrink `components/TemplateCustomizer.tsx` from 1122 lines to ~200 lines.
**Strategy:** Extract customizer controls by semantic sections, isolate individual field components, and separate token mutation logic into a standalone hook.

```text
src/features/applications/detail/components/
├── TemplateCustomizer.tsx (Main orchestrator, ~200 lines)
└── TemplateCustomizer/
    ├── sections/
    │   ├── FitSection.tsx
    │   ├── ThemeSection.tsx
    │   ├── TypographySection.tsx
    │   ├── LayoutSection.tsx
    │   └── SectionSpacingOverrides.tsx
    ├── fields/
    │   ├── SegmentedField.tsx
    │   ├── SelectField.tsx
    │   ├── RangeField.tsx
    │   ├── ColorField.tsx
    │   └── Headers.tsx (SectionHeading, FieldHeader)
    ├── useTokenPatch.ts (Token patch helper hook)
    ├── types.ts (Explicit prop contracts)
    └── utils.ts (emptyToNull, pruneEmpty, countOverrides, etc.)
```

### 3. PromptEditorCard
**Target:** Shrink `components/PromptEditorCard.tsx` from 829 lines to ~150 lines.
**Strategy:** Separate the heavy data-fetching, validation, and retry state from the component tree.

```text
src/features/applications/detail/components/
├── PromptEditorCard.tsx (Accordion/tabs shell, ~150 lines)
└── PromptEditor/
    ├── usePromptEditorState.ts (API fetch, open, retry, save, rerun, and scope state)
    ├── PromptCategoryEditor.tsx
    ├── PromptField.tsx
    ├── PromptRerunPanel.tsx
    ├── PromptLoadError.tsx
    ├── types.ts
    └── parserUtils.ts (extractPlaceholders, validatePlaceholders, etc.)
```

---

## 🚀 Execution & Rollout Strategy
To maintain a stable `main` branch, we will execute this refactor incrementally. Each of the architectural areas will be addressed in an isolated sequence using the following standard checkout protocol.

For **each extracted module/file**:
1. **Extract & Type**: Move the component/state/utility. Define explicit prop interfaces in the domain's local `types.ts` (Never copy `ReturnType<typeof ...>` into child files).
2. **Update Orchestrator**: Refactor the main file to act purely as the compositional shell.
3. **Gate 1 - Targeted Testing**: Run the domain's specific tests against the refactored module to verify core functionality hasn't degraded.
   * `bun run test -- ApplicationDetailPage.test.tsx`
   * `bun run test -- TemplateCustomizer.test.tsx`
   * `bun run test -- PromptEditorCard.test.tsx`
4. **Gate 2 - Static Analysis & Hygiene**:
   * `bun run --silent check:types`
   * `bun run --silent lint`
   * `bun run lint:deps` *(Ensures Knip catches any stale files or orphaned exports)*
5. **Gate 3 - Full Suite**: `bun run test`
