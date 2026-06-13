Review the SEP at the path given in the user's arguments (e.g. `_docs/sep/12-oneof.md`). If no path is given, ask which SEP to review.

Read the SEP fully. Skim 1–2 nearby SEPs (e.g. `5-runtime-checks.md`, `11-anonymous-structs.md`) and the relevant `compiler/` source for any feature this SEP composes with — enough to evaluate the design in context, not in isolation.

The goal of this review is to evaluate **whether the proposed design and implementation will work** — not whether the document is polished. Ignore:

- Formatting, section ordering, missing-section checklists, template conformance
- Whether vocabulary is defined at first use, whether examples appear early enough, junior-engineer comprehensibility
- Naming-convention adherence, diagnostic-format consistency, mangling-separator correctness
- "Each non-goal should say why deferred" or "each alternative should cite design philosophy" — only flag a missing rationale when the *decision itself* is suspect without it
- Line-number citations and section-by-section checklists

Spend your effort on substance. A polished document proposing a broken design is a worse outcome than a rough document proposing a sound one.

# What to evaluate

## 1. Design soundness

Is the proposed semantic model coherent? Does it compose cleanly with the rest of the language, or does it create surprises?

- **Edge cases the design must handle.** For each new construct, work through: empty / single-element / nested / recursive / null / owned / borrowed / cross-package. If the SEP has not thought through one of these, that's a finding — not because a checklist demands it, but because real users will hit it.
- **Interactions with existing features.** Read what the SEP says about SEP-1 ownership, SEP-3 nullables, SEP-7 classes, SEP-8 interfaces, SEP-10 modules, SEP-11 anonymous structs, etc., *only when the interaction is non-trivial*. Look for unstated interactions and silent assumptions. Example: a new aggregate type that doesn't say how it drops, or that breaks the move-semantics invariant, is a real problem; the same SEP failing to mention SEP-7 when SEP-7 is irrelevant is not.
- **Internal consistency.** Does Decision N contradict Decision M? Does the layout described in one section match the offset assumed by the backend section? Does the AST shape match what the parser would produce? These are the most damaging defects because they survive the document and become bugs.
- **Composability traps.** Features that look local but compose multiplicatively with existing types (nullable × pointer × array × this) often have under-specified composition rules. Identify them.
- **Precedent-setting.** Does this SEP introduce a pattern that future SEPs will inherit (good or bad)? Asymmetries between similar constructs (e.g. `name: T` meaning different things in different contexts) are particularly worth flagging.

## 2. Implementation plan quality

Could a competent engineer build this from the SEP alone, or are there hand-waves that will become design questions later?

- **Layout, sizing, alignment.** Specified concretely (offsets, padding, formulas) or hand-waved? A "tag at offset 0, payload after" without the offset rule is a hand-wave.
- **New IR ops.** Are operands and result types specified? Is the backend lowering at least sketched? Are new ops actually justified, or could existing ones be reused?
- **New types in the type system.** Identity rule (nominal/structural/per-site) stated and consistent? `Equals` semantics defined precisely enough to implement? Cross-form equality (this type vs. sibling types) addressed?
- **Cross-module / mangling.** When the feature crosses package boundaries, is the mangled symbol stable, collision-free, and bounded in length? Recursive cases (nested anonymous types) handled?
- **Drop / lifetime.** Does the SEP say when memory is freed, by what routine, and how owned-pointer payloads are handled?
- **Phases and ordering.** Does the implementation order respect actual dependencies (type registration before field resolution, etc.)? Are there steps that look small but require new infrastructure?
- **Unresolved technical risk.** Anything the SEP defers to "the implementer fills this in" or "TBD" that is actually a load-bearing decision.

## 3. Testing strategy

Will the proposed tests catch real bugs in the implementation?

- **Coverage of the design surface.** Each new semantic rule, IR op, layout invariant, and runtime check needs at least one test that would fail if the implementation got it wrong. Identify gaps.
- **Composition coverage.** Where the feature composes with existing features (oneof × array, oneof × struct field, nullable × this), is at least one combined test specified? Or is composition explicitly declared mechanical?
- **Cross-package tests.** If structural identity, mangling, or cross-module dispatch matter, a `_examples/projects/<feature>/` test is needed.
- **Negative tests.** Every diagnostic claimed in the SEP should have a fixture that triggers it.
- **Testability red flags.** Behavior only observable via disassembly or symbol inspection; behavior dependent on optimization level or undocumented platform layout. Flag these and propose how to test them anyway.
- **Effort estimate.** Straightforward (existing harness covers it), moderate (new fixtures, no new infrastructure), or hard (new test infrastructure, cross-module setup, runtime assertions). One-sentence justification.

## 4. Real-world implications

Step back from the design and ask what it does to the language and its users.

- **Ergonomics.** Is the common case concise? Are required workarounds painful? Are there foot-guns that look fine in isolation but bite when used together?
- **Migration / compatibility.** Does this change behavior of existing code? Reserve a new keyword that could break existing identifiers? Change layout or ABI of existing types?
- **Future flexibility.** Does this design lock in decisions that a future SEP will want to change? Or does it leave room for the deferred work it gestures at?
- **Scope honesty.** Are the non-goals genuinely deferable, or are some of them load-bearing for the feature to be useful in practice?

# Output format

Produce the review as markdown with these top-level sections. Skip a section only if you have nothing substantive to say.

```
## Verdict
<2-4 sentences: what the SEP proposes, is the design sound, what is the single biggest concern>

## Design soundness
<findings on semantics, interactions, internal consistency, composability, precedent>

## Implementation plan
<findings on layout, IR, types, mangling, drop, ordering, unresolved technical risk>

## Testing strategy
<gaps in coverage; effort estimate>

## Real-world implications
<ergonomics, compatibility, future flexibility, scope honesty>

## Top issues to resolve before implementation
<numbered list, ranked by impact. Each item: the problem, why it matters, a concrete proposed resolution>
```

Be unsparing on substance. Be silent on style. If the SEP is sound, say so directly and keep the review short — don't manufacture findings to fill sections.
