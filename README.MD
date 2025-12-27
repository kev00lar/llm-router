# Production-Grade LLM Router

A high-concurrency, resilient gateway designed to manage request distribution, provider failover, and rate management across multiple AI backends.

## 1. Objective
The goal of this system is to act as a critical gateway that handles fragmented AI providers (OpenAI, Anthropic, or local vLLM) with a focus on high performance, resilience, and cloud-native practices.

## 2. System Architecture
The architecture consists of three core layers:
1.**Router Service (Core)**: A high-concurrency Gin-based gateway on port 3000 handling OpenAI-compatible requests.
2.**Management Interface**: An Admin API for dynamic configuration updates (Hot Reloads).
3.**Infrastructure Layer**: Containerized deployment using optimized multi-stage Dockerfiles and Kubernetes manifests.



```mermaid
graph TD
    A[Client Request] -->|/v1/chat/completions| B[Router Core]
    B -->|Rate Limit Check| C{Middleware}
    C -->|Fail 429| A
    C -->|Pass| D[Weighted Selection]
    D -->|80%| E[Mock Provider A]
    D -->|20%| F[Mock Provider B]
    E -->|5xx Error| D
    D -->|Failover/Retry| F
    B -->|Metrics| G[Prometheus /metrics]
    H[Admin API] -->|Update Weights| B