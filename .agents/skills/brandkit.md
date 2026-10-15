# Onboarding Factory visual rules — for agents

Read this guide before creating or changing portal UI, styles, components, layouts,
illustrations, or product themes. It is the source of truth for the portal's visual
identity. Use these rules over framework defaults and earlier design experiments.
Explicit user instructions take precedence.

Scope: the Onboarding Factory portal, including My learning, Foundations, product
navigation, interactive missions, and Content Studio. Product brands may appear
inside their own content area under the product-theme rules below.

## Identity

- Product name: **Onboarding Factory**. The typeset wordmark is `onboarding factory`.
- Purpose: help engineers understand a product, investigate real code, and prepare
  a safe first contribution through practical learning.
- Character: precise, approachable, contemporary, and calm.
- The visual motif is a route with connected checkpoints. Use it in the mark,
  learning paths, and meaningful system flows.
- The approved direction is a light interface with neutral surfaces and one
  **Muted Violet** primary colour. Keep colour concentrated on actions and selection.
- The homepage uses compact horizontal navigation, centered content, generous
  spacing, one prominent next mission, a lightweight learning path, Foundations,
  and product spaces. Preserve this direction rather than returning to a dense
  dashboard with a permanent global sidebar.
- Put a concrete task and learning outcome before the catalogue. Show progress
  toward an investigation or contribution, not just pages opened.
- All UI copy is **English**. Use short, direct sentences and specific action labels.
- Published metrics, customer claims, and source evidence must be real. Clearly
  label illustrative content and progress in prototypes.

## Color — choose the role, not an arbitrary shade

The primary colour is **Muted Violet `#6550B9`**, from Radix Violet 11.
Do not substitute a more saturated purple, blue, green, berry, or aubergine.

| Token | Value | Use for |
| --- | --- | --- |
| `brand` | `#6550B9` | Primary button fills, links, active navigation indicators, focus rings, current checkpoints, and the coloured mark. |
| `brand-hover` | `#654DC4` | Hover on primary fills; Radix Violet 10. |
| `brand-strong` | `#2F265F` | Strong violet text when needed; Radix Violet 12. |
| `brand-soft` | `#EBE7F5` | Selected backgrounds and small tinted fills. Approximately 12% brand over surface. |
| `canvas` | `#FAF9FB` | Main page background. |
| `surface` | `#FDFCFD` | Cards, controls, menus, and dialogs. |
| `surface-subtle` | `#F2EFF3` | Quiet secondary surfaces and hover areas. |
| `diagram-surface` | `#F0EDF2` | The featured mission's system-flow area. |
| `ink` | `#211F26` | Headings and primary text. |
| `muted` | `#65636D` | Secondary text, metadata, and supporting labels on light surfaces. |
| `border` | `#EAE7EC` | Decorative card borders and separators. |
| `control-border` | `#8E8C99` | Controls and meaningful diagram boundaries that need stronger contrast. |
| `on-brand` | `#FDFCFD` | Text and icons on primary fills. |

Use semantic CSS variables, preferably with an `--of-` prefix, instead of scattering
hex values through components. Keep one shared definition of each role.

Colour rules:

- Use a solid brand fill for the main action. Use `brand-soft` for selected items;
  keep their text at `brand` or `brand-strong` with sufficient contrast.
- Keep large page areas neutral. Do not tint the whole interface violet.
- Use `ink` and `muted` for running text. Reserve violet text for emphasis with a
  functional purpose: a link, selected step, or short brand phrase.
- No blue or green in the portal's decorative palette or primary UI. Product
  branding is a scoped exception, not a new portal theme.
- Never communicate state through colour alone. Pair it with a label, icon, or shape.
- Do not reduce text opacity to create hierarchy. Use the defined text roles.
- Target WCAG 2.2 AA: at least 4.5:1 for normal text, 3:1 for large text, and 3:1 for
  meaningful control and diagram graphics against adjacent colours. Decorative
  separators may remain subtle. Check the actual foreground/background pair.
- The approved primary button pair (`on-brand` on `brand`) is approximately 6.03:1.
  Recheck contrast when introducing another background or changing a state.

### Semantic states

| Meaning | Treatment |
| --- | --- |
| Completed / successful | Check icon and explicit text; neutral or violet. No green is required. |
| Current / selected | Brand indicator or fill, plus an active-state label or accessible state. |
| Upcoming / unavailable | Neutral marker and explicit label. |
| Needs review / potentially outdated | Restrained amber text `#825115` on `#FBF1DF`, plus a warning icon or label. |
| Error / destructive | Red text `#B42335` on `#FEF2F2`, plus an explicit message. |

Warning and error colours express meaning, not decoration. Keep review, publication,
source freshness, and learner progress distinct. A published revision may need an
update; a completed lesson does not establish that its sources are current.

## Typography

- UI family: `'Geist', Arial, sans-serif`. Use Geist for headings, body, and controls.
- Monospace: `'Geist Mono', ui-monospace, monospace` for code, identifiers, step
  numbers, and short eyebrows. Do not use it for body copy or long headings.
- Use weights 400 for body, 500 for headings and controls, and 600 for limited emphasis.
- Use sentence case. Proper product names keep their own spelling. Short eyebrows
  may be uppercase with approximately `0.075em` tracking.
- Keep supporting labels at least 12px. Do not shrink text to fit a crowded layout.
- Use tabular figures for aligned numeric data, not for all editorial text.

| Role | Reference size and treatment |
| --- | --- |
| Homepage heading | 49px desktop; 35–38px on narrow screens; weight 500; leading 1.12; tracking around `-0.06em`. |
| Featured mission heading | 28–31px; weight 500; leading 1.2; tracking around `-0.04em`. |
| Section heading | 16–18px; weight 500. |
| Body / lesson text | 14–16px; leading 1.6–1.8; comfortable reading width. |
| Button / navigation text | 13–14px; weight 500 where emphasis is needed. |
| Metadata / eyebrow | 12px minimum. |

Use smaller product headings in dense tools such as Studio. The larger homepage
scale is not a default for every panel. Reflow content before reducing type sizes.

## Logo files

The mark is an orthogonal route with two circular endpoints. The wordmark uses
`onboarding` in Geist 600 and `factory` in Geist 400 on one line. A mark beside this
wordmark is permitted. Keep the name readable and the route geometry consistent.

Canonical mark geometry:

```svg
<svg viewBox="0 0 32 36" fill="none" xmlns="http://www.w3.org/2000/svg">
  <path d="M6 29V15h20V5" stroke="currentColor" stroke-width="2.7"
        stroke-linejoin="round" />
  <circle cx="6" cy="29" r="3.4" fill="var(--of-surface, #FDFCFD)"
          stroke="currentColor" stroke-width="2.2" />
  <circle cx="26" cy="5" r="3.4" fill="var(--of-surface, #FDFCFD)"
          stroke="currentColor" stroke-width="2.2" />
</svg>
```

The circle fill must match the logo's actual flat background. Use CSS `currentColor`
for the stroke. In code, hide a decorative mark from assistive technology when an
adjacent name already labels it; give a standalone logo link an accessible name.

This guide does not ship separate logo, font, or favicon assets. When adding them,
use explicit filenames such as:

```text
onboarding-factory-mark-currentcolor.svg
onboarding-factory-mark-violet.svg
onboarding-factory-mark-ink.svg
onboarding-factory-mark-white.svg
onboarding-factory-lockup-violet.svg
onboarding-factory-mark-violet-transparent-512.png
```

Keep SVG masters; export PNG only when required. Keep asset and font licences with
bundled files. Do not claim an asset exists until it has been created and inspected.

## Logo usage rules

- Clear space: at least half the mark's height on every side of the logo composition.
- Use a 28–32px mark in the main header; 24px is the normal minimum in controlled UI.
- Keep the typeset wordmark around 16px, with a 14px minimum. Use the mark alone
  when the full name cannot fit without wrapping or becoming unreadable.
- Check a 16px favicon separately for legibility instead of blindly shrinking the mark.
- Keep violet, monochrome ink, and an inverted light version available when needed.
  On a light surface the default is the violet mark with an ink wordmark.
- Scale proportionally. Do not rotate, stretch, add shadows, or redraw the route
  differently on each page. Keep both endpoint circles open and readable.
- On a busy background, provide a flat area or an opaque asset with safe padding.

## UI composition and interactions

- Use a 4px spacing grid. Keep related items close and leave more space between sections.
- Center the desktop homepage content at roughly 1,200–1,300px including padding;
  use about 40px side padding on desktop and 16–24px on narrow screens.
- Use one prominent featured mission, an interactive flow alongside it, then the
  learning path and the Foundations/product sections. Stack naturally on mobile.
- Prefer spacing and typography to repeated card borders. Use roughly 8px corners
  for controls, 12px for small cards, and 14–18px for the main featured panel.
- Give a section one clear primary action. Secondary actions use text or quiet controls.
- Aim for at least 44px touch targets; primary buttons are about 46px tall. Provide
  visible keyboard focus and meaningful accessible names and selected states.
- Keep interactions useful: selecting a flow node reveals context; a mission action
  opens its content. Do not add decorative controls or invented dashboard metrics.
- Use a consistent outline icon family and stroke weight. Keep arrows, nodes, and
  diagram edges simple. Labels must explain meaning without colour alone.
- Keep motion brief and optional; respect reduced motion. Avoid looping decoration.
- Represent generated lesson content as data rendered through shared components.
  New products must not require cloned screens or a different lesson renderer.

## Product themes

The portal owns global navigation, page backgrounds, typography, component geometry,
focus behaviour, mission controls, and semantic state meanings.

A product may supply its own logo, cover, illustrations, and local accents within
its product content area. Use an explicitly applicable product UI kit. A marketing
brand guide does not automatically define the product application's UI.

- Product themes are configuration, not product-specific frontend implementations.
- Product colours must not leak into the portal shell, Foundations, or Studio controls.
- Keep shared lesson structure and interaction behaviour consistent across products.
- Preserve third-party logo proportions and colours; do not recolour them to violet.
- When no product kit is available, use the portal defaults and a clearly generic
  product initial or icon. A placeholder is not an official product logo.

## Anti-patterns — avoid these

1. Reopening the primary-colour decision or replacing Muted Violet with a framework
   default without an explicit user request: it breaks the agreed identity.
2. Blue/green portal accents, whole-page colour washes, gradients, neon, or glow:
   they compete with the restrained light direction.
3. Returning to a dense homepage sidebar/dashboard or boxing every row:
   the approved hierarchy relies on space and one focused next task.
4. Tiny labels, faint text, opacity-based hierarchy, or colour-only status:
   they make the learning experience harder to read and navigate.
5. Heavy shadows, glass effects, 3D decoration, or gratuitous animation:
   they add visual weight without helping the task.
6. Renaming controls, recolouring statuses, or changing interaction patterns per product:
   learners should not have to relearn the portal when switching products.
7. Invented source references, review states, accuracy percentages, or customer claims:
   show actual evidence and label prototypes as illustrative.
8. Copying Antares's marketing colours, type rules, or logo restrictions into this UI:
   Antares is a structural reference for this guide, not the portal's identity.

## References

- [Antares agent guide](https://getantares.io/brand/agents.md): reference for this document's structure; not an inherited portal theme.
- [Radix Colors](https://www.radix-ui.com/colors/docs/palette-composition/scales): Violet and Mauve scales. Primary shades verified against `@radix-ui/colors` 3.0.0.
- [Geist](https://vercel.com/font): UI and monospace font families.
- [WCAG 2.2](https://www.w3.org/TR/WCAG22/): accessibility requirements; the guide is not a claim of complete conformance.
