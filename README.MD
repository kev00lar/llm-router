# AI-Powered LLM Router

A **high-performance, resilient LLM routing gateway** built in **Go**.  
This project exposes a **unified OpenAI-compatible API** that intelligently routes requests across multiple LLM providers with **rate limiting**, **weighted load balancing**, and **automatic failover**.

---

## Key Features

- **OpenAI-Compatible API** (`/v1/chat/completions`)
- **Weighted Load Balancing** across multiple providers
- **Automatic Failover** on provider errors (5xx)
- **Rate Limiting** with HTTP `429` rejection
- **Prometheus Metrics** for observability
- **Admin Interface** for runtime configuration
- **Docker & Kubernetes Ready**

---

##  Architecture Overview

The router sits between clients and multiple LLM backends, acting as a smart gateway that enforces limits, routes traffic, and ensures resiliency.

### Request Lifecycle Diagram

```mermaid
graph TD
    subgraph Client_Layer [Client Layer]
        User[External Client]
    end

    subgraph Router_Service [Router Core - Port 3000]
        RC[Router Core]
        MW{Rate Limit Middleware}
        WS[Weighted Selector]
        MET[Prometheus Metrics]
    end

    subgraph Management_Interface [Admin Interface]
        AA[Admin UI/API]
        DB[(In-Memory State)]
    end

    subgraph Backend_Providers [LLM Backends]
        P1[Mock Provider A]
        P2[Mock Provider B]
    end

    %% Path 1: Rate Limit Rejection
    User -->|POST /v1/chat/completions| RC
    RC --> MW
    MW -- |Tokens Exhausted| --> UserRej[HTTP 429]
    UserRej --> User
    MW -.-> |Record 429| MET

    %% Path 2: Happy Flow
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

    %% Management Flow
    AA -->|Update Weights| DB
    DB -.->|Shared State| WS
```

## Local Development (Docker Compose)

Run the full environment locally, including the router and two mock LLM providers.

### Prerequisites

- Docker & Docker Compose
- Go 1.24+ (only required if running outside Docker)

### Steps

#### Initialize Go Modules

go mod tidy

docker-compose up --build

## Kubernetes Deployment

Deploy the router to a local Kubernetes cluster using **Minikube** or **Kind**.

### Steps

#### Point Docker to the Cluster

eval $(minikube docker-env)
Build the Image
docker build -t llm-router:latest .
Apply Kubernetes Manifests
kubectl apply -f k8s/

Access the Service
minikube service llm-router-service


