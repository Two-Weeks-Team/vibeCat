# PRD Index

## Core

- `LIVE_AGENTS_PRD.md`: development purpose, business model, core diagram, mandatory requirements

## Details

- `DETAILS/IMPLEMENTATION_REQUIREMENTS.md`: technical requirements and acceptance criteria
- `DETAILS/IMPLEMENTATION_STATUS_MATRIX.md`: feature scope and verification evidence matrix
- `DETAILS/TDD_VERIFICATION_PLAN.md`: Red-Green-Refactor implementation and test order
- `DETAILS/IMPLEMENTATION_EXECUTION_PLAN.md`: immediate build order, module dependencies, completion gates
- `DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md`: task-by-task implementation guide from startup menu to full runtime
- `DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`: menu behavior, runtime flows, operational scenarios
- `DETAILS/ASSET_MIGRATION_PLAN.md`: asset copy structure, inventory, and usage rules
- `DETAILS/BACKEND_ARCHITECTURE.md`: backend service architecture, ADK agent graph, Firestore schema, observability
- `DETAILS/CLIENT_BACKEND_PROTOCOL.md`: WebSocket and REST protocol specification, authentication, error codes
- `DETAILS/BACKEND_IMPLEMENTATION_TASKS.md`: backend implementation tasks T-100 to T-146
- `DETAILS/DEPLOYMENT_AND_OPERATIONS.md`: GCP deployment, observability, security, operations
- `DETAILS/SUBMISSION_AND_DEMO_PLAN.md`: submission artifacts, demo flow, verification checklist
- `DETAILS/SOURCE_REFERENCE_MAP.md`: official source links and local paths
- `DETAILS/CLOUDBUILD_SPEC.md`: Cloud Build YAML specs for both backend services

## Character System

- `Assets/Sprites/{name}/preset.json`: voice, size, and persona config per character
- `Assets/Sprites/{name}/soul.md`: character personality prompt (injected server-side)
