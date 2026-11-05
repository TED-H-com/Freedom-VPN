# Project introduction design

This is Freedom VPN's public project introduction, not the member or admin app.
Traditional Chinese is the default at `/Freedom-VPN/`; English has an equivalent
static page at `/Freedom-VPN/en/`. Both work without JavaScript.

## Existing design context

Follow `docs/portal/DESIGN.md` and `apps/portal/src/styles.css`: system fonts,
blue `--primary` actions, slate text, white surfaces and pale slate backgrounds.
The portal's info palette explains the architecture and readiness boundaries.
Reuse its 8px surface radius, 6px control radius and 4/8/16px spacing rhythm.
There are no external fonts, trackers, scripts, or runtime dependencies.

The public AI-Intune and bat-cowork pages inform the bilingual introduction,
source links, architecture explanation and candid status language. Their palettes,
claims and client-side language switching do not carry over.

## Layout and type

Use a 1200px content grid. This marketing surface extends the portal scale with
40–60px display text, 28–40px section titles, 20px subheads, 16–18px main prose
and 14px diagram labels/navigation. CJK display letters retain natural spacing.
At 800px the hero becomes one column; at 520px features and document rows stack.
Keep readable line lengths and at least 44px link targets. Focus is a visible
blue outline; reduced-motion preference disables smooth anchor scrolling.

The architecture figure is the focal point. It is semantic HTML, with separately
labelled control, coordination, node execution and member traffic layers. It is
an explanatory schematic, never a product screenshot or live health indicator.
Capabilities use ruled editorial rows. No statistics, testimonial filler, fake
status dashboards, gradients, icon grids or nested cards.

## Content and maintenance

Keep both HTML files equivalent in scope, headings, documentation destinations,
status limits and metadata. Use real relative links for language switching and
absolute GitHub `blob/main` links for documentation. English points to
`README.md` and `docs/DELIVERY.en.md`; Traditional Chinese points to
`README.zh-TW.md` and `docs/DELIVERY.md`. Technical guides keep their existing
repository language. The product specification and delivery guide govern claims.

## Verification and publishing

Check both languages at 1440, 768 and 390px, keyboard navigation, in-page anchors,
language links with JavaScript disabled, missing assets, and horizontal overflow.
The Pages workflow stages only the HTML, CSS and SVG assets from this directory;
this design note is not published. Deploy only from `main` to `github-pages`.
