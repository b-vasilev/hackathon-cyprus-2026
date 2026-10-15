# users
Goal: explain substantial actor/access detail only when it cannot fit naturally in overview.
Question: Which evidenced roles interact with the product, and what access boundaries apply?
Inputs: shared actor/capability objects, authentication/authorization behavior, user/operator docs, relevant scenario claims and precise gaps.
Investigation: distinguish goals, caller type, authentication and authorization; check enforcement/caller behavior rather than deriving roles from UI labels. Separate end users, operators, administrators and external clients only when supported.
Output: normally integrate actor/outcome summary into overview. If a distinct chapter is useful, write cited Markdown and a concise role/access table using shared identities; a focused context slice is optional if it answers a different question.
Evidence/gaps: label documented roles versus implemented access checks. Preserve unknown permissions/user needs; do not invent customers, demographics, adoption or usage statistics.
Exclusions: no persona portraits, full scenario_ref, copied context diagram or capability inventory.
Acceptance: roles have sourced goals/access boundaries, actor identity matches overview/flows, and a separate chapter adds useful detail without repeating the overview.
