# RootCause: The Anti-Hallucination Architecture

**Built for the micro1 Frontier Engineering Challenge 2026**

RootCause is an agentic workflow that diagnoses performance bottlenecks in backend services. But unlike most AI diagnostic tools, **it is mathematically incapable of hallucinating a root cause.**

It uses a **Detective/Verifier architecture**:
1. **The Detective (LLM)** looks at the raw OpenTelemetry metrics and proposes a hypothesis.
2. **The Verifier (Deterministic Go code)** runs a strict delta-based signature check against a healthy baseline. 
3. If the signature fails, the LLM is rejected, fed the exact failing condition, and forced to try again.

## Why this is necessary

LLMs are people-pleasers. If you give them a dashboard with a mild GC spike and ask "why is the service slow?", they will blame the GC because they want to give you an answer. Real observability requires an agent that is perfectly comfortable saying "none of the math adds up." 

By forcing the LLM to submit its hypothesis to deterministic Go code, we guarantee that the final report is backed by actual signal, not just a plausible-sounding guess.

## The Eval: Proving the Delta

We built a deterministic synthetic dataset of 13 scenarios (11 specific bottlenecks, 1 clean, 1 hard multi-signal case). Each scenario injects the true root cause, but also injects **deliberate noise** (e.g. mildly elevated disk I/O during a lock contention issue).

The eval harness pits a naive **Baseline** (single-shot LLM with no verification) against the **Agent** (Detective/Verifier loop).

## How to Reproduce

Everything is built in standard Go. No external agent frameworks, no databases.

1. **Clone the repo**
2. **Generate the dataset** (this will create `data/scenarios/` and `data/ground_truth.json`):
   ```bash
   go run cmd/generate/main.go
   ```
3. **Verify the signatures mathematically** (this proves exactly 1 signature passes per scenario, without any LLM involved):
   ```bash
   go run cmd/verify/main.go
   ```
4. **Run the full evaluation** (requires Anthropic API key):
   ```bash
   export ANTHROPIC_API_KEY="sk-ant-..."
   go run cmd/eval/main.go
   ```

The eval will output a complete report showing how many times the baseline was fooled by the noise, and how the Agent's rejection loop forced it to find the true cause.

## Outputs
- **`data/eval_results.json`**: The final scored comparison between Baseline and Agent.
- **`data/trajectories/`**: The raw LLM conversation logs showing exactly how the Verifier rejected the LLM and forced it to self-correct.
