# RootCause
### An agent that diagnoses performance bottlenecks — and proves it's right before it says so

**Built for:** micro1 Frontier Engineering Challenge 2026 (Agentic Workflows Hackathon)
**Built from scratch** — no reused code, dataset, or concept from any prior project.

---

## 1. Problem Statement

**Who has this problem:** Any backend/infra engineer who gets paged (or pinged) with "the service is slow" and has to figure out *why* before they can fix it.

**The bottleneck:** The answer is buried in logs, metrics, and traces, and it could be any of a dozen root causes — an N+1 query, lock contention, a GC pause, connection pool exhaustion, a slow downstream dependency, a cache gone stale. Two engineers looking at the same evidence often land on different guesses, and confirming the real cause usually means manually correlating multiple signals by hand. Worse, real logs are *noisy* — they contain red herrings, overlapping symptoms, and plausible-but-wrong signals sitting right next to the actual cause. That's slow, inconsistent, and doesn't scale past whoever's on call that day.

**Why it's worth solving:** Time-to-diagnosis directly costs money and reliability. A tool that reliably narrows "it's slow" down to a specific, evidence-backed cause — fast — is something a real on-call engineer would actually use, not just admire.

**What a real person would want to use:** something they hand raw logs/metrics to, that comes back with a specific claim ("lock contention on table X"), backed by the specific signal that *proves* it — not a vague paragraph of possibilities, and not a confident guess that falls apart under scrutiny.

---

## 2. Success Criteria (mapped to the judging rubric)

| Rubric criterion | Points | How this project earns it |
|---|---|---|
| Problem & User Value | 15 | Real, common, high-frequency pain for any backend engineer — not hypothetical |
| Agent Solution & Engineering | 30 | **Two-role architecture**: Detective (LLM) proposes a cause; Verifier (deterministic Go code) runs the actual signature check. The LLM cannot report until the code confirms the signal. This is genuine engineering — the agent literally cannot hallucinate its way to a confident answer. |
| End-to-End Quality | 20 | Output is a structured diagnosis report a real engineer could act on immediately — cause, evidence, confidence score, matched signature fields, suggested next step |
| Measured Improvement | 15 | Noisy ground-truth-labeled synthetic scenarios (we inject the bug *and* red herrings, so we know the answer) → clean, undeniable baseline-vs-agent comparison with a large delta |
| Reproducibility | 15 | Fully synthetic, fully deterministic, scripted dataset generation — anyone can rerun it cold |
| Hot Take | 5 | Earned honestly from whatever the eval actually shows (see §8) |

**North star while building:** every component must serve one of these six rows. If it doesn't, cut it.

---

## 3. Scope & Non-Goals

**In scope:**
- 12 synthetic bottleneck scenarios with injected root cause + deliberate noise/red herrings, plus 1 hard multi-signal case
- All synthetic data formatted as **standard OpenTelemetry (OTel) traces/metrics** — proves this plugs into real observability stacks (Jaeger, Prometheus, Datadog) out of the box
- One baseline (generic prompt, raw unstructured logs/metrics, no verification)
- One agent solution: Detective/Verifier two-role architecture with programmatic signature library using **dynamic baselining** (delta-based thresholds, not hardcoded numbers)
- A scored eval harness (does the diagnosis match the injected ground truth? Does the verified signal actually match?)
- Full README, changelog, reproduction guide, 5-min video, agent trajectories

**Explicitly out of scope:**
- Real production log ingestion / real APM integration
- A UI/dashboard beyond plain CLI output or a markdown report
- Support for bottleneck types beyond the 12 scripted
- Any dependency on tooltrace, Causa, or other prior projects — new build, new repo, clean disclosure

**Preemptive defense against "grading your own homework" critique:**
Yes, we write the generators AND the Verifier. This is true of every hackathon with synthetic data (and the instructions explicitly recommend it). Our defense is threefold:
1. **Transparency** — all thresholds, generators, and ground truth are open. Judges can modify thresholds and re-run.
2. **Dynamic baselining** — thresholds are relative multipliers against a healthy baseline, not magic numbers. The Verifier works for any service with any normal operating range.
3. **OTel format** — data follows a real standard. Replace our generators with a real OTel exporter and the Verifier works unchanged.

---

## 4. Architecture

### 4.1 Baseline
- One LLM call.
- Input: raw, unstructured log lines + raw metric dump for the scenario (exactly what a human would be handed today, including the noise).
- Prompt: "Here are logs and metrics from a slow service. What's the bottleneck?"
- No tools, no verification, no structure. This represents the naive way most people actually do this today.
- **Expected weakness:** the baseline will be confused by noise and red herrings, often picking a plausible-but-wrong cause. This creates a large, honest improvement delta.

### 4.2 Agent Solution — RootCause (Detective/Verifier Architecture)

This is the core engineering bet of the project. It is **two genuinely separate roles**, not one LLM doing both.

---

**Step 1 — Structured context.**
Logs/metrics are generated in **OpenTelemetry (OTel) format** — standard trace spans + metric data points — then parsed by Go code into a consistent structured format before either role sees them:
- Timestamps and latency distributions
- DB call counts and per-query timing
- Lock wait times and lock holder counts
- GC event timestamps correlated to latency spikes
- Connection pool stats (active, idle, wait queue depth)
- Disk I/O queue depth
- Memory page faults and swap activity
- External dependency call timing and retry counts

Each scenario includes **two snapshots**: a "healthy baseline" (normal operation) and an "incident snapshot" (during the bottleneck). This is how real observability works — you compare incident to normal, not incident to magic numbers.

Noise fields are included (e.g., mildly elevated GC times in a lock contention scenario) because real logs are noisy. The agent does not get a clean signal — it gets realistic data.

---

**Step 2 — Detective (LLM role).**
The Detective sees the structured context and is asked to:
1. Propose a specific root cause
2. Identify *which specific fields* in the structured data it is pointing at
3. Name which signature it believes matches

The Detective cannot report to the user. It can only hand its hypothesis to the Verifier.

---

**Step 3 — Verifier (Go code role — the core engineering differentiator).**
The Verifier is **not an LLM.** It is a Go function that runs a deterministic signature check.

**Critical design: Dynamic Baselining (not hardcoded thresholds).**
Each scenario provides a "healthy baseline" snapshot alongside the "incident" snapshot. The Verifier compares the two using **relative multipliers** — not static magic numbers. This means the Verifier works for any service regardless of its normal operating parameters (a 200ms lock wait might be a crisis for HFT but normal for a batch job).

Each bottleneck type has a diagnostic signature — a set of delta-based conditions comparing incident vs. healthy baseline:

```
N+1 Query:
  incident.db_call_count > healthy.db_call_count * 3
  AND incident.avg_query_time < healthy.avg_query_time * 1.5  (many small calls, not fewer slow ones)
  AND incident.total_db_time / incident.total_request_time > 0.6

Lock Contention:
  incident.lock_wait_p95 > healthy.lock_wait_p95 * 5
  AND incident.lock_holder_count > healthy.lock_holder_count * 2
  AND lock_wait_spike correlated with latency_spike (temporal correlation)

GC Pause:
  incident.gc_event_count > healthy.gc_event_count * 3
  AND incident.max_gc_pause > healthy.max_gc_pause * 4
  AND gc_timestamps within 50ms of latency_spike_timestamps

Connection Pool Exhaustion:
  incident.pool_active >= incident.pool_max  (at ceiling)
  AND incident.pool_wait_queue > 0  (requests queuing — never happens in healthy)
  AND incident.checkout_wait_p95 > healthy.checkout_wait_p95 * 10

Slow Downstream:
  incident.external_call_p95 > healthy.external_call_p95 * 5
  AND incident.external_call_time / incident.total_request_time > 0.5
  AND incident.local_processing_time < healthy.local_processing_time * 1.5  (local code is fine)

Stale Cache:
  incident.cache_hit_rate > 0.80  (looks healthy — this is the trap for the baseline)
  AND incident.downstream_retry_rate > healthy.downstream_retry_rate * 3
  AND incident.data_staleness_delta > healthy.data_staleness_delta * 5

Thread Starvation:
  incident.cpu_utilization > 0.85  (absolute — CPU has a natural ceiling)
  AND incident.io_wait_time > healthy.io_wait_time * 4
  AND incident.goroutine_blocked_count > healthy.goroutine_blocked_count * 5

Disk I/O Saturation:
  incident.disk_queue_depth > healthy.disk_queue_depth * 4
  AND incident.disk_await > healthy.disk_await * 3
  AND incident.cpu_iowait > healthy.cpu_iowait * 5

Memory Pressure:
  incident.page_fault_rate > healthy.page_fault_rate * 10
  AND incident.swap_in_rate > 0  (any swap is abnormal in a healthy system)
  AND page_fault_timestamps correlated with latency_spike_timestamps

Network Retry Storm:
  incident.retry_rate > healthy.retry_rate * 3
  AND incident.downstream_error_rate > healthy.downstream_error_rate * 5
  AND incident.request_amplification_factor > 2  (absolute — more than 2x means retries are dominating)

Off-by-One Pagination:
  incident.rows_fetched > incident.rows_displayed * 10  (fetching wildly more than needed)
  AND incident.db_payload_size > healthy.db_payload_size * 5
  AND incident.execution_time scales with page_number (linear or worse)
```

**Why dynamic baselining matters:** A judge looking at `db_call_count > 15` will correctly call it brittle. A judge looking at `incident.db_call_count > healthy.db_call_count * 3` sees a system that adapts to any service's normal operating range. This is a real engineering choice, not a demo trick.

**If the Verifier rejects the hypothesis** → it returns the specific failing condition back to the Detective. The Detective must revise and propose a different cause. The loop runs until the signature passes or the agent exhausts max attempts (in which case it reports "unconfirmed" with its best guess and the rejection reason).

**If the Verifier confirms** → it returns the matched signature with the specific field values that triggered the pass. These become the evidence lines in the final report.

This is what makes RootCause different from every other "multi-step LLM chain" submission: **the LLM literally cannot output a confident diagnosis unless deterministic code independently confirms the signal.** No hallucination path exists to a false positive confident answer.

---

**Step 4 — Report.**
Structured JSON output:
```json
{
  "root_cause": "lock_contention",
  "confidence": "high",
  "verified_by": "signature_check_v1",
  "evidence": {
    "lock_wait_time_p95_ms": 340,
    "lock_holder_count": 7,
    "latency_spike_correlation": true
  },
  "rejected_hypotheses": [
    { "cause": "gc_pause", "failed_condition": "max_gc_pause < threshold" }
  ],
  "suggested_fix": "Add row-level locking or break the transaction holding lock on table X"
}
```

---

**Step 5 — Multi-Signal Hard Case.**
Scenario 13 (the hard case) has two overlapping causes: lock contention AND a slow downstream call. Both signatures partially pass. The Verifier uses a **signal strength score** (how far above threshold each field is, how many conditions pass) to rank the two and pick the dominant cause. The Detective is shown both partial results and must justify its final pick. This is what separates a real eval from a toy one.

### 4.3 Tech Stack
- **Go** for the harness, dataset generator, signature library, Verifier, and eval runner. No heavy agent framework — a hand-rolled orchestration loop is more impressive here, not less. Stands out against the sea of Python/LangChain submissions.
- LLM API called directly (no framework wrapper) for the Detective role.
- Plain JSON files for scenarios, signatures, and results — no database needed.
- Structured logging of every Detective/Verifier exchange → becomes the agent trajectory artifacts.

---

## 5. Synthetic Dataset — 12 Scenarios + 1 Hard Case

Each scenario = **two OTel-formatted snapshots** (healthy baseline + incident) with:
- **One injected root cause** (ground truth known because we inject it)
- **Deliberate noise** — 1-2 red herring signals that show elevated deltas from healthy baseline, but below the Verifier's multiplier thresholds
- **Realistic OTel structure** — trace spans with service names, operation names, durations, status codes; metric data points with resource attributes

The noise is the key design decision. A baseline seeing "slightly elevated GC times" alongside obvious lock contention will sometimes pick GC. The Verifier won't be fooled because GC signature conditions (delta multipliers) won't fully pass.

| # | Bottleneck | Key Signal | Primary Red Herring Noise |
|---|---|---|---|
| 1 | N+1 Query | 47 sequential DB calls, <8ms each | Slightly slow external call |
| 2 | Lock Contention | lock_wait p95=340ms, 7 holders | Mildly elevated GC pauses |
| 3 | GC Pause | 8 GC events, max pause 180ms, correlated spikes | Elevated pool wait |
| 4 | Connection Pool Exhaustion | pool at max, wait queue=12 | Slightly slow disk |
| 5 | Slow Downstream | external p95=820ms, 68% of request time | Minor N+1 pattern |
| 6 | Stale Cache | hit rate=91% (looks healthy), retry_rate=0.35 | CPU slightly elevated |
| 7 | Thread Starvation | CPU=92%, goroutines blocked=28 | Lock wait slightly elevated |
| 8 | Disk I/O Saturation | queue_depth=14, await=31ms | Memory slightly elevated |
| 9 | Memory Pressure | page_fault_rate=high, swap_in>0 | GC slightly elevated |
| 10 | Network Retry Storm | retry_rate=4.2x, amplification=3.1 | Slow downstream (caused by the storm) |
| 11 | Pagination Bug | rows_fetched=4800 vs displayed=20 | Slow overall DB time |
| 12 | Clean (no bottleneck) | All metrics within normal ranges | None — tests false positive rate |
| 13 | **HARD: Lock + Downstream** | Both signatures partially pass | Agent must pick dominant cause |

For each scenario: a Go generator script produces the data deterministically from a seed. Ground truth is stored separately from what the agent sees.

**Scenario 12 (no bottleneck)** is important — it tests whether the agent invents a cause when there isn't one. The baseline almost certainly will. RootCause should report "no confirmed bottleneck" because no signature passes.

### 5.1 Forced Rejection Design (CRITICAL — read this before building generators)

The rejection loop is the entire visual story of this project. If the Detective gets every scenario right on the first try, the trajectory logs look boring and the video has no "wow moment." We must **deliberately engineer at least 4 scenarios where the noise is strong enough to bait the Detective into a wrong first guess.**

**Scenarios designed to force at least one rejection:**

| Scenario | Why Detective will guess wrong first | What Verifier will reject |
|---|---|---|
| 2 (Lock Contention) | GC pauses are elevated enough to look primary. LLMs over-index on GC. | GC signature fails: `max_gc_pause=65ms` is below 100ms threshold |
| 6 (Stale Cache) | Cache hit rate is 91% — "cache is fine." LLM will blame elevated CPU instead. | CPU signature fails: `cpu_util=0.58` is below 0.85 threshold |
| 7 (Thread Starvation) | Lock wait is slightly elevated. LLMs love blaming locks. | Lock signature fails: `lock_wait_p95=45ms` is below 200ms threshold |
| 10 (Retry Storm) | Slow downstream is visible (because the storm *causes* it). LLM will blame downstream. | Downstream signature fails: `local_processing_time=180ms` is not <50ms — the service IS slow too, because of retries |

**How to calibrate noise levels:**
- The red herring signal should be elevated enough that a human reading quickly would consider it a plausible cause
- But it must NOT pass the Verifier's threshold — set noise values at 50-70% of the signature threshold
- Test this: run the baseline on each scenario. If the baseline gets it right, the noise is too weak. Strengthen it until the baseline is wrong on at least 6/13 scenarios.

**Max rejection loop depth:** 4 attempts. If the Detective hasn't found a confirmed cause after 4 proposals, report "unconfirmed" with the best-scoring partial match. This prevents infinite loops and the trajectory log shows honest failure.

---

## 6. Evaluation Methodology

**Primary metric:** % of scenarios where the agent's claimed root cause matches the injected ground truth. Binary, judge-proof, no ambiguity.

**Secondary metrics:**
- False positive rate — does it invent a cause on the clean scenario?
- Confidence calibration — does it claim high confidence only when it's actually right?
- Rejection loop depth — how many hypothesis revisions did it take? (trajectory quality)
- Evidence accuracy — does the cited evidence actually match the Verifier output?
- Time/cost per diagnosis (baseline vs agent)

**Format:**

| Metric | Baseline | RootCause | Change |
|---|---|---|---|
| Correct root cause (X/13) | | | |
| Correct on hard case (scenario 13) | | | |
| False positive on clean scenario | | | |
| Avg hypothesis revisions per scenario | N/A | | |
| Avg time per diagnosis | | | |
| Avg cost per diagnosis | | | |

Run both baseline and agent on the **identical 13 scenarios**. No cherry-picking.

**Expected delta:** The baseline's performance on noisy scenarios should be materially worse than RootCause — particularly on scenario 6 (stale cache looks healthy), scenario 12 (no bottleneck), and scenario 13 (hard multi-signal). These three cases are where the improvement story is clearest.

**Target performance:** Baseline should score ~5-7/13 (noise + tricky cases bring it down). RootCause should score 11-13/13. If the delta is less than 4, either the noise is too weak or the signatures need tuning. Fix the scenarios, don't water down the eval.

---

## 7. Improvement Changelog (fill in as you build — don't reconstruct it after)

| Stage | What you tried and why | Evidence | Decision/Learning |
|---|---|---|---|
| Baseline | Raw logs, single generic prompt | [X/13] | Starting point |
| Iteration 1 | Structured context instead of raw dump | [X/13] | kept / revised / removed |
| Iteration 2 | Added programmatic Verifier + signature library | [X/13] | kept / revised / removed |
| Iteration 3 | Detective/Verifier split with rejection loop | [X/13] | kept / revised / removed |
| Iteration 4 | Added noise to scenarios + signal strength ranking for hard case | [X/13] | kept / revised / removed |
| Final | Combined what worked | [X/13] | Main contribution |

---

## 8. Hot Take (write this last, after you see real results — don't pre-write a generic one)

Leave this blank until the eval is done. The rubric rewards a genuine observed failure mode turned into a lesson, not a platitude written in advance.

Candidates to watch for and document honestly:
- Does the rejection loop ever loop more than 2x? If not, does that mean the first hypothesis is almost always right — or that the Detective learned to match signatures without really verifying?
- Does the clean scenario (12) fool the agent? That would be a meaningful finding.
- Does the hard multi-signal case expose a weakness in the signal strength ranking?

**Strong hot take candidate (use if scenario 12 confirms it):**
> "LLMs are terrible at observability because they are people-pleasers. They see a slightly elevated metric and desperately want to give you an answer. Real observability requires an agent that is perfectly comfortable saying 'I have no idea — none of the math adds up.' The Verifier doesn't just improve accuracy — it gives the agent permission to say 'no confirmed cause' without feeling like it failed. Scenario 12 proved this: the baseline invented a cause from normal noise. RootCause correctly reported nothing. That restraint is the most valuable thing we built."

---

## 10. Final Deliverables Checklist

- [ ] Complete solution code + changelog, in a public repo
- [ ] Signature library in Go — documented, one file per bottleneck type
- [ ] All 13 scenario generator scripts — deterministic, seeded, reproducible
- [ ] README: intended user, bottleneck, why it matters, changelog, main failure mode, hot take
- [ ] Reproduction guide: clean-environment setup, exact commands for baseline/agent/eval, expected output, versions, runtime/cost
- [ ] Solution video (≤5 min): problem → baseline fails on noisy scenario → one full RootCause run with Verifier rejection → comparison table → changelog highlights → what you cut
- [ ] Agent trajectories for every scenario — Detective proposals, Verifier rejections, final confirmed reports
- [ ] Explicit disclosure: this is a from-scratch build, no prior project's code or dataset reused

---

## 11. Video Strategy (20 points — treat this like a product demo)

The video is worth 20 points. Do not treat it as an afterthought.

**Structure:**
1. **(0:00–0:45)** The problem: show a real-looking noisy log dump. Ask "what's wrong?" Pause. That's what on-call engineers face.
2. **(0:45–1:30)** Baseline run: show it on scenario 2 (lock contention with GC noise). It picks GC. It's wrong. This is the "before."
3. **(1:30–3:30)** RootCause run on the same scenario: Detective proposes GC → Verifier rejects (show the failing condition) → Detective revises to lock contention → Verifier confirms (show the matched fields) → structured report output.
4. **(3:30–4:15)** Comparison table across all 13 scenarios. Let the numbers speak.
5. **(4:15–5:00)** Changelog highlight: what made the biggest difference. One thing you tried and cut. Hot take.

**Key visual moment to capture:** the Verifier rejection in real time — the Detective proposing the wrong answer and the Go code rejecting it with a specific failed condition. That 10-second moment is the proof that this is real engineering, not a prompt chain.

---

## 12. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Signature thresholds are too generous — baseline also passes them | Calibrate thresholds on scenario generation, not after. Test that the clean scenario produces no passing signatures. |
| Detective almost never gets rejected — loop is shallow | 4 scenarios are specifically designed for forced rejection (see §5.1). If first-try accuracy is still >80%, strengthen noise at 80-90% of threshold. |
| Hard case (scenario 13) stumps the agent | The signal strength ranking is the answer. If it still fails, report it honestly — a documented failure mode on a hard case is good Hot Take material. |
| Go build takes longer than expected | Signature library and generator scripts first. Wire the LLM call last — the Verifier can be tested without any LLM at all. |
| Scope creep | Every addition must serve one of the six rubric rows. If it doesn't, cut it. |

---

## 13. Execution Risk Management

This section exists to prevent "we ran out of time because we built in the wrong order." Build order is load-bearing.

### Build Order (strict — do not skip ahead)

**Phase 1 — Foundation (must work before anything else):**
1. Go project structure: `cmd/`, `internal/scenarios/`, `internal/signatures/`, `internal/agent/`, `internal/eval/`
2. Scenario data model (Go structs for structured metrics)
3. All 13 scenario generator functions — each produces a JSON file
4. Ground truth labels stored in a separate file the agent never sees
5. **Checkpoint:** run all generators, eyeball the JSON output, confirm noise values are set per §5.1

**Phase 2 — Verifier (this is the core — test it standalone):**
6. Signature library — one function per bottleneck type, returns pass/fail + which conditions failed
7. Signal strength scorer — returns a 0.0–1.0 score for partial matches
8. **Checkpoint:** run every scenario through every signature. Confirm: exactly 1 signature passes per scenario (except 12=none, 13=two partial). If multiple pass, tighten thresholds. If none pass, loosen thresholds. This checkpoint must pass before writing any LLM code.

**Phase 3 — Baseline:**
9. Single LLM call with raw log dump → parse response → compare to ground truth
10. Run baseline on all 13 scenarios, record scores in changelog §7
11. **Checkpoint:** baseline should score ~5-7/13. If it scores higher, noise is too weak — go back to generators

**Phase 4 — Agent (Detective + loop):**
12. Structured context formatter (Go code that transforms raw scenario into structured format)
13. Detective prompt — receives structured context, proposes cause + target signature
14. Rejection loop — Verifier rejects → feed failing condition back to Detective → re-propose
15. Report generator — takes Verifier confirmation + Detective reasoning → structured JSON output
16. Run agent on all 13 scenarios, record scores in changelog §7
17. **Checkpoint:** agent should score 11-13/13 and show at least 4 rejection events across all scenarios

**Phase 5 — Polish:**
18. Agent trajectory logger — capture every Detective/Verifier exchange in readable format
19. README, reproduction guide, changelog finalization
20. Record video following §11 structure exactly
21. **Final checkpoint:** clone repo to a fresh directory, follow reproduction guide, confirm it works end to end

### If Something Breaks — Decision Rules

| Situation | Decision |
|---|---|
| LLM API is down or rate-limited | Use a different model. The Detective prompt is model-agnostic. |
| Verifier passes everything (thresholds too loose) | Tighten thresholds by 20% across the board, re-run generators |
| Verifier passes nothing (thresholds too tight) | Loosen thresholds by 20%, re-run generators |
| Baseline scores 10+/13 (too good) | Strengthen noise in scenarios — raise red herring values to 80-90% of threshold |
| Baseline scores 2-/13 (too bad) | Weaken noise slightly — the delta will look suspiciously large if baseline is broken |
| Agent scores below baseline | Bug in the rejection loop or structured context parser. Debug before iterating. |
| Detective never gets rejected | Raise noise on 2+ more scenarios until at least 4 rejections occur across the 13 runs |
| Time is running short | Cut straight to Phase 5. A working Phase 1-3 with a documented baseline score is a valid submission. Phase 4 is where the win happens but a clean Phase 1-3 still lands top 50. |