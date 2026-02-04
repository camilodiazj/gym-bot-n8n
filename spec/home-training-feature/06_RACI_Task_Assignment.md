# RACI Matrix - Home Training Feature Implementation

## Team Roles

| Role | Responsable | Descripción |
|------|-------------|-------------|
| **n8n-agent** | n8n Expert | Modificaciones a workflows, system prompts, nodos |
| **pixel-dev** | Developer | Cambios en BD, migraciones, lógica de código |
| **kiro-coach** | Training Expert | Validación de reglas de entrenamiento, compensaciones |
| **code-reviewer** | QA Analyst | Testing, validación, criterios de aceptación |

---

## RACI Legend

- **R** = Responsible (Ejecuta la tarea)
- **A** = Accountable (Responsable final)
- **C** = Consulted (Debe ser consultado)
- **I** = Informed (Debe ser informado)

---

## Phase 1: KYC/Encuesta Modifications

| Tarea | n8n-agent | pixel-dev | kiro-coach | code-reviewer |
|-------|-----------|-----------|------------|---------------|
| Modificar FormPrompt.txt (FASE 6.5) | **R/A** | I | C | I |
| Agregar campos a Tool_Create_User_Profile | **R/A** | C | I | I |
| Definir preguntas de equipamiento | C | I | **R/A** | I |
| Validar flujo de conversación | R | I | C | **A** |
| Test de KYC con usuario HOME | R | I | I | **R/A** |

---

## Phase 2: GymRatForm Modifications

| Tarea | n8n-agent | pixel-dev | kiro-coach | code-reviewer |
|-------|-----------|-----------|------------|---------------|
| Modificar ProcessUserPreferences (Code) | **R/A** | C | I | I |
| Implementar parseHomeEquipment() | **R** | **A** | I | C |
| Modificar query GetExercisesByPattern | **R/A** | C | I | I |
| Actualizar RoutineCreation.txt | **R** | I | **A** | C |
| Definir reglas de compensación de gaps | C | I | **R/A** | I |
| Definir mapeo equipamiento español→canónico | R | I | **A** | I |
| Test de generación rutina HOME | R | I | C | **R/A** |

---

## Phase 3: Database & Templates

| Tarea | n8n-agent | pixel-dev | kiro-coach | code-reviewer |
|-------|-----------|-----------|------------|---------------|
| Agregar columna environment a routine_templates | I | **R/A** | I | I |
| Crear 15 templates HOME | I | **R** | **A** | C |
| Definir day_requirements HOME | C | **R** | **A** | I |
| Definir sets ajustados por patrón | I | R | **A** | I |
| Ejecutar migraciones | I | **R/A** | I | I |
| Validar datos en BD | I | R | I | **A** |

---

## Phase 4: Testing & QA

| Tarea | n8n-agent | pixel-dev | kiro-coach | code-reviewer |
|-------|-----------|-----------|------------|---------------|
| Crear usuarios de prueba | I | **R** | I | **A** |
| Escribir test cases | I | I | C | **R/A** |
| Ejecutar E2E tests en n8n | **R** | I | I | **A** |
| Validar cobertura muscular | I | I | **R/A** | C |
| Validar equipamiento filtrado | R | I | I | **R/A** |
| Regression testing (usuarios GYM) | R | I | I | **R/A** |
| Sign-off final | I | I | C | **R/A** |

---

## Task Checklist by Role

### n8n-agent Tasks

- [ ] **P1-N8N-01**: Modificar FormPrompt.txt - Agregar FASE 6.5
- [ ] **P1-N8N-02**: Agregar 2 campos en Tool_Create_User_Profile
- [ ] **P2-N8N-01**: Modificar ProcessUserPreferences - parseHomeEquipment()
- [ ] **P2-N8N-02**: Modificar query GetExercisesByPattern con filtro equipment
- [ ] **P2-N8N-03**: Actualizar system prompt RoutineCreation.txt
- [ ] **P4-N8N-01**: Ejecutar E2E tests para casos HOME

### pixel-dev Tasks

- [ ] **P3-DEV-01**: Migration: Agregar columna environment a routine_templates
- [ ] **P3-DEV-02**: INSERT: 15 nuevos templates HOME
- [ ] **P3-DEV-03**: INSERT: day_requirements para templates HOME
- [ ] **P3-DEV-04**: Crear usuarios de prueba HOME
- [ ] **P3-DEV-05**: Validar integridad de datos post-migration

### kiro-coach Tasks

- [ ] **P1-KC-01**: Definir lista de equipamiento válido para casa
- [ ] **P2-KC-01**: Definir reglas de compensación por gap
- [ ] **P2-KC-02**: Validar mapeo equipamiento español→canónico
- [ ] **P3-KC-01**: Definir min_sets ajustados para cada patrón HOME
- [ ] **P3-KC-02**: Aprobar templates HOME (validación científica)
- [ ] **P4-KC-01**: Validar cobertura muscular de rutinas generadas

### code-reviewer Tasks

- [ ] **P1-QA-01**: Validar flujo de conversación KYC
- [ ] **P2-QA-01**: Code review de ProcessUserPreferences
- [ ] **P2-QA-02**: Code review de queries SQL
- [ ] **P4-QA-01**: Escribir test cases detallados
- [ ] **P4-QA-02**: Ejecutar regression tests (usuarios GYM)
- [ ] **P4-QA-03**: Validar criterios de aceptación
- [ ] **P4-QA-04**: Sign-off final

---

## Dependencies

```
┌─────────────────────────────────────────────────────────────────┐
│                      DEPENDENCY GRAPH                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Phase 1 (KYC)                                                 │
│  ├─ P1-N8N-01 (FormPrompt) ──────────────────┐                │
│  └─ P1-N8N-02 (Tool fields) ─────────────────┼──→ P1-QA-01    │
│                                               │                │
│  Phase 2 (GymRatForm)                        │                │
│  ├─ P2-KC-01 (Gap rules) ────────────────────┼──→ P2-N8N-03   │
│  ├─ P2-N8N-01 (ProcessUserPrefs) ────────────┼──→ P2-N8N-02   │
│  └─ P2-N8N-02 (Query filter) ────────────────┴──→ P2-QA-01    │
│                                                                 │
│  Phase 3 (Database)                                            │
│  ├─ P3-DEV-01 (Column) ──→ P3-DEV-02 (Templates)              │
│  ├─ P3-KC-01 (Sets) ─────→ P3-DEV-03 (day_reqs)               │
│  └─ P3-DEV-02 ───────────→ P3-KC-02 (Approval)                │
│                                                                 │
│  Phase 4 (Testing) - Depends on P1, P2, P3 completion          │
│  ├─ P3-DEV-04 (Test users) ──→ P4-QA-01 (Test cases)          │
│  ├─ P4-N8N-01 (E2E) ─────────→ P4-QA-03 (Acceptance)          │
│  └─ P4-KC-01 (Validation) ───→ P4-QA-04 (Sign-off)            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Estimated Effort by Phase

| Phase | n8n-agent | pixel-dev | kiro-coach | code-reviewer | Total |
|-------|-----------|-----------|------------|---------------|-------|
| Phase 1 | 4h | 1h | 2h | 2h | **9h** |
| Phase 2 | 6h | 2h | 4h | 3h | **15h** |
| Phase 3 | 1h | 6h | 4h | 2h | **13h** |
| Phase 4 | 3h | 2h | 3h | 6h | **14h** |
| **Total** | **14h** | **11h** | **13h** | **13h** | **51h** |

---

## Communication Channels

| Tipo | Canal | Frecuencia |
|------|-------|------------|
| Daily standup | Slack #gymbot-home-feature | Diario |
| Blockers | Slack (tag @team) | Inmediato |
| Code reviews | GitHub PRs | Por commit |
| Design decisions | Documento de spec | Según necesidad |

---

## Definition of Done

Una tarea se considera **DONE** cuando:

1. ✅ Código/configuración implementado
2. ✅ Revisado por el rol designado (C o A en RACI)
3. ✅ Tests pasan (si aplica)
4. ✅ Documentación actualizada
5. ✅ Marcado como completado en este checklist
