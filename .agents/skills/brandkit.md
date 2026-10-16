# Wayfinder visual rules — for agents

Read this guide before creating or changing portal UI, styles, components, layouts,
illustrations, or product themes. It is the source of truth for the portal's visual
identity. Use these rules over framework defaults and earlier design experiments.
Explicit user instructions take precedence.

Scope: the Wayfinder portal, including My learning, Foundations, product
navigation, interactive missions, and Content Studio. Product brands may appear
inside their own content area under the product-theme rules below.

## Identity

- Product name: **Wayfinder**. The typeset wordmark is `Wayfinder`. Always use this exact spelling and casing in the UI, page titles, and accessible names.
- Purpose: help engineers understand a product, investigate real code, and prepare
  a safe first contribution through practical learning.
- Character: friendly, calm, encouraging, and clear. The visual direction is
  called **Quest**: learning that feels like progress, without fake gamification.
  It is light and flat: muted violet, thin rounded borders and light type weights.
- The visual motif is a route with connected checkpoints. Use it in the mark,
  learning paths, and meaningful system flows.
- The approved direction is a light interface with white surfaces, one **Quest
  Violet** primary colour and one **Coral** accent for progress and achievements.
  Keep colour concentrated on actions, selection and progress.
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

The primary colour is **Quest Violet `#664CCD`**, a muted violet. The accent is **Coral `#FF6A3D`**,
used only for progress fills and achievements. Do not substitute another purple,
blue, green, berry, or aubergine.

| Token | Value | Use for |
| --- | --- | --- |
| `brand` | `#664CCD` | Primary button fills, links, eyebrows, focus rings, current checkpoints, and the coloured mark. |
| `brand-hover` | `#765ED4` | Hover on primary fills. |
| `brand-strong` | `#3C2D76` | Strong violet text, such as selected navigation and tab labels. |
| `brand-soft` | `#F0EEF9` | Selected backgrounds and small tinted fills. |
| `accent` | `#FF6A3D` | Progress bar fills and the unlocked-achievement border. Graphics only, never text. |
| `accent-soft` | `#FFE8DF` | Achievement icon backgrounds. |
| `accent-strong` | `#B33D16` | Achievement icons and any coral-family text on `accent-soft`. |
| `canvas` | `#FFFFFF` | Main page background. |
| `surface` | `#FFFFFF` | Cards, controls, menus, and dialogs. |
| `surface-subtle` | `#F5F4F9` | Quiet secondary surfaces, hover areas and progress tracks. |
| `diagram-surface` | `#F7F7FB` | Route panels and diagram areas. |
| `ink` | `#1F1B32` | Headings and primary text. |
| `muted` | `#605C70` | Secondary text, metadata, and supporting labels. |
| `border` | `#E9E7F0` | Card borders and separators. |
| `border-strong` | `#D5D1E5` | Route connectors and stronger dividers. |
| `control-border` | `#8983A0` | Inputs, checkpoints and meaningful diagram boundaries that need 3:1 contrast. |
| `on-brand` | `#FFFFFF` | Text and icons on primary fills. |

Use semantic CSS variables, preferably with an `--of-` prefix, instead of scattering
hex values through components. Keep one shared definition of each role.

Colour rules:

- Use a solid brand fill for the main action. Use `brand-soft` for selected items;
  keep their text at `brand` or `brand-strong` with sufficient contrast.
- Use coral only for progress and achievements. It does not replace violet for
  actions, and white text on coral does not meet the text contrast target.
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
- The approved primary button pair (`on-brand` on `brand`) is approximately 6.0:1.
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

- UI family: `'Lexend', Arial, sans-serif` (`--of-font`). Use Lexend for headings,
  body, controls and eyebrows. It is wide and highly legible, so give labels room
  rather than shrinking them.
- Monospace: `'DM Mono', ui-monospace, monospace` (`--of-font-mono`) for code,
  identifiers and step numbers. Do not use it for body copy or long headings.
- Keep the type light. Use the weight tokens, never literal numbers:
  `--of-weight-regular` (300) for body text, `--of-weight-medium` (400) for labels
  and navigation, `--of-weight-semibold` (500) for buttons, `h3`, eyebrows and
  `strong`, and `--of-weight-bold` (600) for `h1` and `h2`.
- Use sentence case. Proper product names keep their own spelling. Eyebrows are
  uppercase Lexend 500 in `brand`, with approximately `0.08em` tracking.
- Keep supporting labels at least 12px. Do not shrink text to fit a crowded layout.
- Use tabular figures for aligned numeric data, not for all editorial text.

| Role | Reference size and treatment |
| --- | --- |
| Homepage heading | 49px desktop; 35–38px on narrow screens; weight 600; leading 1.12; tracking around `-0.035em`. |
| Featured mission heading | 28–31px; weight 600; leading 1.2; tracking around `-0.03em`. |
| Section heading | 18–22px; weight 600. |
| Body / lesson text | 14–16px; leading 1.6–1.8; comfortable reading width. |
| Button / navigation text | 13–14px; buttons weight 500, navigation weight 400. |
| Metadata / eyebrow | 12px minimum. |

Use smaller product headings in dense tools such as Studio. The larger homepage
scale is not a default for every panel. Reflow content before reducing type sizes.

## Logo files

The approved mark is a W-shaped route with two open circular endpoints. It combines
Wayfinder's initial with the route motif. The wordmark is `Wayfinder` in Lexend 500
on one line. Keep the name readable and the route geometry consistent.

Canonical mark geometry:

```svg
<svg viewBox="0 0 40 40" fill="none" stroke="currentColor"
     xmlns="http://www.w3.org/2000/svg">
  <path d="M5 12L11 31L20 17L29 31L35 12" stroke-width="3"
        stroke-linejoin="round" />
  <circle cx="4" cy="9" r="3" stroke-width="2.2" />
  <circle cx="36" cy="9" r="3" stroke-width="2.2" />
</svg>
```

The shortened route stops at the endpoint rings, keeping their centres transparent.
Do not fill the circles to match a particular background. Use `currentColor` for
inline SVG strokes. An SVG inside an `<img>` does not inherit CSS colour: use the
fixed violet, ink, or white file. Hide a decorative mark from assistive technology
when an adjacent name labels it; give the home link an accessible name.

The site ships only four image files:

- `frontend/public/brand/wayfinder-mark-violet.svg`: transparent mark for the header.
- `frontend/public/brand/wayfinder-lockup-violet.svg`: violet mark with an ink
  wordmark; Lexend 500 is embedded so it works without an installed font.
- `frontend/public/favicon.svg`: white W on a violet tile.
- `frontend/public/favicon.ico`: 16/32/48px browser fallbacks in one file.

`frontend/public/brand/Lexend-OFL.txt` covers the embedded font. See the adjacent
`README.md` for usage. The header uses the SVG mark and live Lexend 500 text.

The favicon omits the open endpoint rings and uses rounded stroke ends to keep
the W clear at 16px; this is the approved small-size variant. Create other colors,
PNG exports, or home-screen icons only when there is a concrete need.

Keep the SVG masters and font licence with the assets. Do not claim an asset
exists until it has been created and inspected.

## Logo usage rules

- Clear space: at least half the mark's height on every side of the logo composition.
- Use a 38px mark in the main header; 24px is the normal minimum in controlled UI.
- Keep the header wordmark at 19px, with a 14px minimum elsewhere. Use the mark alone
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
- Use the radius tokens: `--of-radius-control` (12px) for buttons, inputs and tabs,
  `--of-radius-card` (18px) for cards, and `--of-radius-panel` (24px) for the
  featured mission, route and other large panels.
- Surfaces are flat. Cards, quiz choices, inputs and panels use a thin 1px rounded
  border and no shadow or raised edge. On hover a card's border changes colour.
- Primary buttons are a flat `brand` fill; secondary buttons a 1px
  `control-border` outline on `surface`.
- Navigation items and page tabs are pills: `brand-soft` fill with `brand-strong`
  text when active. Diagram tab groups use a solid `brand` fill for the active tab.
- Route checkpoints are 30px numbered circles joined by a 3px line. The current
  checkpoint is a solid violet circle with a `brand-soft` halo; a completed one
  shows a check.
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

1. Reopening the primary-colour decision or replacing Quest Violet with a framework
   default without an explicit user request: it breaks the agreed identity.
2. Blue/green portal accents, coral used for actions or text, whole-page colour
   washes, gradients, neon, or glow: they compete with the light direction.
3. Returning to a dense homepage sidebar/dashboard or boxing every row:
   the approved hierarchy relies on space and one focused next task.
4. Tiny labels, faint text, opacity-based hierarchy, or colour-only status:
   they make the learning experience harder to read and navigate.
5. Drop shadows, raised or pressed 3D edges, thick borders, heavy type, glass
   effects, or gratuitous animation: they add visual weight without helping the task.
6. Renaming controls, recolouring statuses, or changing interaction patterns per product:
   learners should not have to relearn the portal when switching products.
7. Invented source references, review states, accuracy percentages, or customer claims:
   show actual evidence and label prototypes as illustrative.
8. Copying Antares's marketing colours, type rules, or logo restrictions into this UI:
   Antares is a structural reference for this guide, not the portal's identity.

## References

- [Antares agent guide](https://getantares.io/brand/agents.md): reference for this document's structure; not an inherited portal theme.
- [Lexend](https://www.lexend.com/): UI font family, designed for reading proficiency.
- [DM Mono](https://fonts.google.com/specimen/DM+Mono): monospace family.
- [WCAG 2.2](https://www.w3.org/TR/WCAG22/): accessibility requirements; the guide is not a claim of complete conformance.
