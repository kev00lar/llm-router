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
    subgraph Client_Layer [Client Layer]
        User[External Client]
    end

    subgraph Router_Service [Port 3000]
        RC[Router Core]
        MW{Rate Limit Middleware}
        WS[Weighted Selector]
        MET[Prometheus Metrics]
    end

    subgraph Backend_Providers [LLM Backends]
        P1[Mock Provider A]
        P2[Mock Provider B]
    end

    %% Path 1: Rate Limit Rejection
    User -->|POST /chat| RC
    RC --> MW
    MW -- |Tokens Exhausted| --> UserRejection[HTTP 429]
    UserRejection --> User
    MW -.-> |Record 429| MET

    %% Path 2: Happy Flow (Provider A)
    MW -- |Tokens Available| --> WS
    WS -->|Primary Attempt| P1
    P1 -- |Success 200| --> RC
    RC -.-> |Record Success A| MET
    RC -->|Return Response| User

    %% Path 3: Resilient Flow (Failover)
    P1 -.->|5xx Error| WS
    WS -.-> |Record Error A| MET
    WS -->|Retry| P2
    P2 -- |Success 200| --> RC
    RC -.-> |Record Success B| MET