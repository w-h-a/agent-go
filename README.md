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
        Context[Context Window]
    end
    
    subgraph Brain [Model Layer]
        LLM[LLM Interface]
    end
    
    subgraph State [Memory Layer]
        ShortTerm[Session Memory]
        LongTerm[Vector Store]
    end
    
    subgraph Actions [Tool Layer]
        Catalog[Tool Catalog]
        UTCP[UTCP Client]
    end
    
    UAgent --> Orchestrator
    Orchestrator -->|1. Retrieve| State
    Orchestrator -->|2. Plan| LLM
    Orchestrator -->|3. Execute| Actions
    Actions -->|4. Result| Orchestrator