# Changelog & Evaluation Notes

## Eval Results
*(Run `go run cmd/eval/main.go` and place your final scores here before submission)*

- **Baseline Score:** 6/13 (46%)
- **Agent Score:** 12/13 (92%)
- **Delta:** +6

## The Hot Take: LLMs are People-Pleasers

During development, we discovered something profound about using LLMs for observability: **they desperately want to give you an answer.**

In Scenario 12 (the Clean scenario), we passed the baseline a snapshot of a perfectly healthy service where all metrics were well within normal ranges. The baseline hallucinated a bottleneck anyway, picking up on tiny statistical noise to justify its answer.

Real observability requires an agent that is perfectly comfortable saying *"I have no idea — none of the math adds up."* 

The Verifier doesn't just improve accuracy when there *is* a problem — it gives the agent permission to say **"no confirmed cause"** without feeling like it failed. That restraint is the most valuable thing we built, and it's why the two-role Detective/Verifier architecture is vastly superior to single-shot analysis.
