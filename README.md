# RootCause: Deterministic Verification for AI-Assisted Observability

Built for the micro1 Frontier Engineering Challenge 2026.

## 1. Context & Value Proposition

**Intended User:** Site Reliability Engineers (SREs), DevOps professionals, and backend infrastructure teams performing incident triage.

**Current Bottleneck:** When a production incident occurs, engineers face a wall of metrics and logs. LLMs can process these metrics quickly, but they suffer from high hallucination rates in observability contexts. In our evaluation, a single-shot LLM diagnosed the wrong root cause 54% of the time, often blaming statistically irrelevant noise (like a minor GC spike) simply because it appeared elevated.

**Why Solving It Is Valuable:** False positives in incident response waste critical triage time and erode trust in AI-assisted tooling. By forcing an AI diagnostic agent to mathematically prove its hypothesis against deterministic delta-based signatures, we eliminate diagnostic hallucinations, reduce false escalations, and lower Mean Time to Resolution (MTTR).

## 2. Architecture

RootCause separates hypothesis generation from validation using a **Detective/Verifier** loop:

1. **The Detective (LLM):** Ingests OpenTelemetry-formatted baseline and incident snapshots. Uses Anthropic Tool Use (`tool_choice: {type: "tool", name: "submit_diagnosis"}`) to output a strict JSON hypothesis containing the proposed cause, a 0–1 confidence score, specific metric deltas as evidence, and explicitly ruled-out alternatives.
2. **The Verifier (Deterministic Go):** Evaluates the LLM's hypothesis against hardcoded delta-based signatures (e.g., `lock_wait_p95 must exceed 5x baseline`). If the signature fails, the LLM is rejected with the exact failing condition and forced to retry.
3. **Dominant-Cause Resolution:** When a hypothesis passes verification, the Verifier also scores all other passing signatures using a normalized overshoot function. If a stronger signal exists, the LLM's pick is rejected in favor of the dominant cause. This handles multi-signal incidents where multiple bottlenecks overlap.

### Determinism Boundaries

To be precise about what is and is not deterministic in this system:

- **Deterministic:** Synthetic data generation, signature verification logic, signal-strength scoring, and the evaluation harness. Running `go run cmd/generate/main.go` and `go run cmd/verify/main.go` will produce identical results on any machine.
- **Non-deterministic:** The LLM hypothesis generation depends on a live Anthropic API call (model: `claude-haiku-4-5`). LLM outputs are inherently stochastic, so exact scores may vary slightly across runs. The Verifier compensates for this by rejecting incorrect hypotheses regardless of how confidently they are stated.

## 3. Reproduction Guide

Written for someone starting from a clean environment.

### Prerequisites
- Go 1.21 or higher
- An Anthropic API Key (model used: Claude 4.5 Haiku, `claude-haiku-4-5`)
- **Approximate runtime:** ~2 minutes for a full 13-scenario evaluation
- **Approximate cost:** ~$0.10 USD

### Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/tijani-web/root-cause.git
   cd root-cause
   go mod tidy
   ```

2. **Configure your API Key:**
   Create a `.env` file in the project root:
   ```env
   ANTHROPIC_API_KEY=sk-ant-...
   # Required only for Identity-Linked API keys:
   ANTHROPIC_WORKSPACE_ID=wrk_...
   ```

### Running the Solution

3. **Generate the synthetic dataset** (13 scenarios + ground truth):
   ```bash
   go run cmd/generate/main.go
   ```

4. **Validate the deterministic signatures** (proves verification math is sound before the LLM is involved):
   ```bash
   go run cmd/verify/main.go
   ```
   Expected: All 13 scenarios report `[PASS]`.

5. **Run the full evaluation** (Baseline LLM vs. Detective/Verifier Agent):
   ```bash
   go run cmd/eval/main.go
   ```

### Expected Output
- **Console:** Real-time log of LLM hypotheses and Verifier rejections, ending with a scored comparison.
- **`data/eval_results.json`:** The full scored comparison matrix with reasoning for every scenario.
- **`data/trajectories/`:** Raw LLM conversation logs for each scenario, showing the exact multi-turn feedback loops between the Detective and Verifier.

### Relevant Versions
- **Go:** 1.21+
- **Model:** `claude-haiku-4-5` (Anthropic Claude 4.5 Haiku)
- **Anthropic API version:** `2023-06-01`
