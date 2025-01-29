# tmpl

## 0.10.0

### Minor Changes

- bb3c432: Drop RenderAssociated and AssociatedTemplate in favour of unified Tmpl function
- bb3c432: Change Template interface to return single value
- bb3c432: Simplify templates layout structure using slots

## 0.9.0

### Minor Changes

- 3254a91: Rename lazy template func to tmpl

## 0.8.0

### Minor Changes

- c2dec08: Add lazy & slot template funcs for slotted content support

### Patch Changes

- c2dec08: Pass reference to the executing template to context template funcs

## 0.7.0

### Minor Changes

- 9d88623: Add clsx template func

## 0.6.4

### Patch Changes

- bbf0801: Revert trimming func input string
- bbf0801: Add vite_dev template func
- 8a470a3: Allow using default sync renderer without explicitly creating a renderer

## 0.6.3

### Patch Changes

- b9dc768: Fix LoadTree skipping pages when there are no layouts

## 0.6.2

### Patch Changes

- a875fef: Trim funcs input string

## 0.6.1

### Patch Changes

- ea58d68: Load names template after the last file

## 0.6.0

### Minor Changes

- 5a5e249: Add Vite support
- fe20cc4: Add HTML Streaming support

## 0.5.0

### Minor Changes

- d1af662: Rename SetLayoutDir to SetLayoutRoot

## 0.4.0

### Minor Changes

- 90c5f3c: Rename templatesParser.LoadDir to templatesParser.LoadWithLayouts
  Merge Combine and NewTemplate to one API and rename to Tmpl
  Add AssociatedTemplates
  Update docs
  Add tests

## 0.3.0

### Minor Changes

- 2bfc93e: Update tmpl.Combine API

## 0.2.1

### Patch Changes

- 6bc368f: Update docs

## 0.2.0

### Minor Changes

- 90a257c: Add publish script

## 0.1.0

### Minor Changes

- 027b9c2: Initial API
