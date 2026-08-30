# RootCause: Deterministic Verification for AI Observability

Built for the micro1 Frontier Engineering Challenge 2026.

## 1. Context & Value Proposition

**Intended User:** Site Reliability Engineers (SREs), DevOps professionals, and backend infrastructure teams.

**Current Bottleneck:** When a production incident occurs, engineers face a wall of metrics and logs. LLMs can process these metrics, but they suffer from high hallucination rates in observability—often blaming statistically irrelevant noise (like a minor Garbage Collection spike) simply because it stands out.

**Why Solving It Is Valuable:** False positives in incident response waste critical triage time. By forcing an AI diagnostic agent to mathematically prove its hypothesis against deterministic signatures, we can eliminate diagnostic hallucinations, reduce false escalations, and drastically lower Mean Time to Resolution (MTTR).

## 2. Architecture

RootCause separates hypothesis generation from validation using a **Detective/Verifier** architecture:
1. **The Detective (LLM):** Ingests OpenTelemetry-formatted baseline and incident snapshots. It utilizes Anthropic Tool Use to output a strict JSON hypothesis containing the proposed cause, confidence score, and specific metric deltas.
2. **The Verifier (Deterministic Go):** Evaluates the LLM's hypothesis against hardcoded delta signatures (e.g., locking wait times must exceed 5x the baseline). If the signature fails, the LLM is rejected and forced to retry with the exact failing condition.

## 3. Reproduction Guide

This guide is designed for a clean environment. 

### Prerequisites
- Go 1.21 or higher
- An Anthropic API Key (Claude 4.5 Haiku is used for the evaluation)
- **Approximate runtime:** ~1 minute
- **Approximate cost:** ~$0.10 for a full 13-scenario evaluation run

### Setup & Data Generation

1. **Clone the repository and install dependencies:**
   ```bash
   git clone https://github.com/tijani-web/root-cause.git
   cd rootCause
   go mod tidy
   ```

2. **Generate the Dataset:**
   This command generates 13 deterministic synthetic scenarios (11 specific bottlenecks, 1 clean state, 1 hard multi-signal state) in `data/scenarios/`. It also generates the ground truth mapping.
   ```bash
   go run cmd/generate/main.go
   ```

3. **Validate the Deterministic Signatures:**
   This proves the underlying Go signatures are mathematically sound before the LLM is introduced.
   ```bash
   go run cmd/verify/main.go
   ```

### Evaluation Run

1. **Configure your API Key:**
   Create a `.env` file in the root directory and add your Anthropic credentials:
   ```env
   ANTHROPIC_API_KEY=sk-ant-...
   # Optional: Required only if you are using an Identity-Linked API key
   ANTHROPIC_WORKSPACE_ID=wrk_...
   ```

2. **Run the Evaluation Harness:**
   This script runs the unassisted Baseline LLM against the 13 scenarios, followed immediately by the Detective/Verifier Agent loop. 
   ```bash
   go run cmd/eval/main.go
   ```

### Expected Output
- The console will output a real-time log of the LLM hypotheses and Verifier rejections, concluding with a final comparison delta.
- **`data/eval_results.json`**: The final scored comparison matrix.
- **`data/trajectories/`**: Raw LLM conversation logs showcasing the exact feedback loops that shaped the agent's final decision.
