# Improvement Changelog

## Final Evaluation Results
- **Baseline Score:** 6/13 (46%)
- **Agent Score:** 12/13 (92%)
- **Delta:** +6

---

## Iteration 1: Synthetic Dataset & Deterministic Signatures
- **Action:** Created a generator for 13 specific performance scenarios (N+1 queries, lock contention, etc.) using an OpenTelemetry-inspired JSON schema. Built a deterministic Go `Verifier` that calculates relative deltas between healthy and incident snapshots.
- **Evidence/Decision:** Hardcoded absolute thresholds failed across different synthetic loads. We pivoted to dynamic baselining (comparing incident metrics to a healthy baseline multiplier) to ensure accurate cross-environment verification.

## Iteration 2: Baseline LLM Evaluation
- **Action:** Implemented a single-shot LLM call (the "Baseline") using a standard diagnostic prompt to analyze the incident snapshots.
- **Evidence/Decision:** The baseline scored 6/13 (46%). It consistently hallucinated causes based on injected red-herring noise (e.g., picking `slow_downstream` when the root cause was actually a stale cache). This proved the absolute necessity of an active validation loop.

## Iteration 3: Detective/Verifier Rejection Loop
- **Action:** Implemented the two-stage agent architecture. The LLM hypothesis is now passed to the Go Verifier. If rejected, the LLM is fed the exact failing condition and forced to reconsider based on the mathematical constraints.
- **Evidence/Decision:** While accuracy improved significantly, the agent struggled to consistently output valid JSON via standard prompt instructions, causing parsing panics in the evaluation harness.

## Iteration 4: Strict Schema via Anthropic Tool Use
- **Action:** Refactored the LLM client to use native Anthropic Tool Use. We enforced a strict JSON schema requiring `proposed_cause`, `confidence`, `evidence` (with metric deltas), and `ruled_out` alternatives.
- **Evidence/Decision:** The Agent scored 12/13 (92%), entirely eliminating JSON parsing errors and forcing the LLM to ground its guesses in cited math. The evaluation trajectories proved the agent successfully self-corrected after Verifier rejections.

---

## Main Failure Mode
In **Scenario 13 (Lock Contention AND Slow Downstream)**, multiple bottleneck signatures overlap heavily. The agent successfully recognized a problem but failed to untangle the compounding metrics, ultimately classifying it incorrectly. Structured context and single-signature validation struggle when multiple distinct faults occur simultaneously.

## The Hot Take
During development, we discovered that LLMs in observability are chronic people-pleasers. Given a clean, healthy dashboard (Scenario 12), the baseline hallucinated a bottleneck anyway just to provide an answer. 

Real observability requires an agent that is perfectly comfortable saying *"none of the math adds up."* The greatest value of the Verifier loop isn't just catching mistakes—it's giving the agent the mathematical permission to safely report **"No Confirmed Cause"** without hallucinating.
