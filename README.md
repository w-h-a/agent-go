# agent

## The Problem

I got tired of building agents via copy pasta.

## The Solution

Agent provides a clean scaffolding for agents where every component is an interface.

## Usage

Coming soon!

## Architecture

Agent acts as the coordinator between the user, the model, memory, and tools.

```mermaid
graph TD
    User[User Input] --> UAgent[Your Agent]
    
    subgraph Agent [Agent]
        Orchestrator[Orchestrator]
    end

    subgraph Generator [Model Layer]
        LLM[LLM Interface]
    end
    
    subgraph State [Memory Layer]
        ShortTerm[Session Memory]
        LongTerm[Vector Store]
    end
    
    subgraph Provider [Tool Layer]
        Catalog[Tool Catalog]
        UTCP[UTCP Client]
    end
    
    UAgent --> Orchestrator
    Orchestrator -->|Store| State
    Orchestrator -->|Plan| Generator
    Orchestrator -->|Execute| Provider
