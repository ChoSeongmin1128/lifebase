---
name: lifebase-api-hexagonal-enforcer
description: Enforce LifeBase API and hexagonal architecture rules. Use when creating or changing endpoints, controllers, use cases, ports/adapters, response contracts, or cross-layer responsibilities.
---

# LifeBase API Hexagonal Enforcer

- Keep `1 endpoint = 1 controller = 1 usecase`.
- Restrict controllers to validation and response mapping.
- Keep business logic in usecase/domain layers only.
- Maintain adapter/port separation for DB and external APIs.
- Verify frontend consumers follow documented architecture references in `docs/400` and `docs/420`.
