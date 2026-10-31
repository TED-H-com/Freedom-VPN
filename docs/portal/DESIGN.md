# Portal design

The portal is a compact operator console and member account interface. Extend the
existing styles in `apps/portal/src/styles.css`; keep observed results separate
from requested changes. zh-TW is the default, with English available separately.

## Palette and typography

Use the existing semantic variables: `--bg-page`, `--bg-surface`, `--bg-subtle`,
`--text-primary`, `--text-secondary`, `--text-muted`, `--border-color`, `--primary`
and the success, warning, danger and info families. Avoid undeclared
`--color-*` variables. Use the system font stack; titles use the existing 1.55rem
scale, tables 0.9rem and badges 0.78rem. Keep long IDs readable and wrapping.

## Components and layout

Reuse `.card`, `.data-table`, `.table-wrapper`, `.form-field`, `.form-grid`,
`.btn-row` and `.badge`. Row actions are secondary buttons; confirmation is the
primary action in a dialog. Status needs text as well as color. Read-only usage
outbox values use a neutral definition list, with no thresholds or alarm colors.
List views need loading, error and empty states. Keep operation receipts visible
while independently checking the affected device.

Use the existing 8px card radius, 6px control radius and 16px spacing rhythm.
Cards have borders; only dialogs need elevation. Native modal dialogs provide
focus containment, Escape cancellation and focus restoration.

## Responsive behavior

Keep tables horizontally scrollable inside their container, without widening
the document. At 768px and below, navigation wraps above content and form grids
become one column. Mobile inputs use 16px text and actions a 44px minimum height.
Verify 390px, 768px and 1440px layouts.

## Extension guide

For a new list, reuse the members table and cursor controls. For a read-only
projection, use labelled values and explicit absent/null states. For an async
action, reuse `OperationBanner` and require resource read-back before claiming
completion. Explain outcomes in UI copy; keep API paths and decision IDs in docs.
