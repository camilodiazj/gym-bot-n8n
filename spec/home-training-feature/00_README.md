# Home Training Feature - Implementation Plan

**Project:** GymBot - Rutinas de Entrenamiento en Casa
**Version:** 1.0
**Status:** Planning
**Created:** 2026-02-03
**Last Updated:** 2026-02-03

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Business Case](#2-business-case)
3. [Scope](#3-scope)
4. [Team Roles & Responsibilities](#4-team-roles--responsibilities)
5. [Timeline Overview](#5-timeline-overview)
6. [Success Metrics](#6-success-metrics)
7. [Risks & Mitigations](#7-risks--mitigations)
8. [Document Index](#8-document-index)

---

## 1. Executive Summary

### Overview

Este proyecto extiende GymBot para soportar **rutinas de entrenamiento en casa**, permitiendo a usuarios sin acceso a gimnasio recibir planes personalizados adaptados a su equipamiento disponible.

### Current State

- GymBot genera rutinas exclusivamente para ambiente **GYM**
- El catálogo de ejercicios contiene 1,657 ejercicios
- Los workflows de n8n (KYC, GymRatForm) asumen entrenamiento en gimnasio
- La tabla `routine_templates` tiene columna `environment` pero solo valor 'GYM'

### Target State

- Soporte completo para entrenamientos en **CASA** y **GYM**
- Flujo KYC adaptado para capturar ambiente de entrenamiento y equipamiento casero
- Motor de selección de ejercicios filtrado por ambiente
- Templates de rutinas específicos para casa con volumen ajustado
- 825 ejercicios disponibles para casa (49.8% del catálogo actual)

### Key Deliverables

1. Modificaciones al flujo KYC para capturar `training_environment` y `home_equipment`
2. Nuevos templates de rutinas para ambiente CASA
3. Lógica de filtrado de ejercicios por ambiente y equipamiento
4. Ajustes de volumen y selección para entrenamientos en casa
5. Manejo de gaps en patrones de movimiento con baja cobertura

---

## 2. Business Case

### Problem Statement

Actualmente, GymBot solo puede generar rutinas para usuarios con acceso a gimnasio. Esto excluye a un segmento significativo de usuarios potenciales que:

- No tienen membresía de gimnasio
- Prefieren entrenar en casa por conveniencia
- Tienen limitaciones de tiempo o movilidad
- Quieren combinar entrenamientos en casa y gimnasio

### Market Opportunity

| Segmento | Potencial |
|----------|-----------|
| Usuarios sin gimnasio | Alto - mercado desatendido |
| Usuarios hibridos (casa + gym) | Medio - feature futura |
| Usuarios viajeros | Medio - rutinas portatiles |

### Value Proposition

- **Para usuarios**: Flexibilidad de entrenar en cualquier lugar
- **Para GymBot**: Mayor base de usuarios potenciales
- **Diferenciador**: Personalización basada en equipamiento real disponible

### ROI Justification

- Incremento estimado de 30-40% en usuarios elegibles
- Reducción de abandono por falta de acceso a gimnasio
- Oportunidad de upselling hacia rutinas de gimnasio premium

---

## 3. Scope

### In Scope

| Area | Descripcion |
|------|-------------|
| **Database** | Nuevas columnas en `users_gym_profile`: `training_environment`, `home_equipment` |
| **KYC Workflow** | Preguntas adicionales para ambiente y equipamiento |
| **GymRatForm Workflow** | Filtrado de ejercicios por ambiente |
| **Exercise Catalog** | Clasificacion de ejercicios existentes por viabilidad en casa |
| **Routine Templates** | Nuevos templates optimizados para casa |
| **Set Profiles** | Ajustes de volumen para entrenamientos en casa |
| **Gap Handling** | Estrategias para patrones con baja cobertura |

### Out of Scope (v1.0)

| Area | Razon |
|------|-------|
| Entrenamientos hibridos (casa + gym) | Complejidad - fase futura |
| Compra de equipamiento recomendado | Feature de e-commerce - fuera de core |
| Videos de ejercicios en casa | Requiere produccion de contenido |
| Rutinas de viaje (hotel) | Variante especifica - fase futura |
| App movil dedicada | El flujo es via WhatsApp |

### Assumptions

1. Los usuarios responderan honestamente sobre su equipamiento
2. El catálogo actual de ejercicios tiene suficiente cobertura para casa
3. Los usuarios de casa aceptarán menor variedad de ejercicios
4. No se requiere nuevo equipamiento especial para n8n

### Dependencies

| Dependencia | Owner | Status |
|-------------|-------|--------|
| Columnas BD agregadas | pixel-dev | Completado |
| Analisis de ejercicios viable | kiro-coach | Completado |
| Credenciales n8n | Infraestructura | Disponible |
| Ambiente de testing | pixel-dev | Requerido |

---

## 4. Team Roles & Responsibilities

### n8n-agent

**Responsabilidad Principal:** Modificaciones a workflows de n8n

| Tarea | Entregable |
|-------|------------|
| Modificar KYC workflow | Nuevas preguntas de ambiente/equipamiento |
| Actualizar GymRatForm | Filtrado de ejercicios por ambiente |
| Ajustar ProcessUserPreferences | Mapeo de equipamiento a filtros |
| Actualizar system prompts | Prompts en espanol para nuevo flujo |

**Archivos a modificar:**
- `n8n/running_flows/GymRatFlow_Supabase_V2_Workout_Tracker.json`
- `n8n/running_flows/GymRatForm Supabase v3.json`
- `n8n/system_prompts/kyc_agent.md`

### pixel-dev

**Responsabilidad Principal:** Cambios en base de datos y backend

| Tarea | Entregable |
|-------|------------|
| Migrations de BD | Nuevas columnas, enums, constraints |
| Routine templates CASA | Nuevos templates en BD |
| Set profiles CASA | Volumenes ajustados para casa |
| Day requirements CASA | Patrones por dia para casa |
| API updates (si aplica) | Endpoints para workout-tracker |

**Tablas a modificar:**
- `users_gym_profile` (ya completado)
- `routine_templates`
- `template_days`
- `day_requirements`
- `set_profiles`
- `exercises` (campo `home_viable`)

### kiro-coach

**Responsabilidad Principal:** Validacion de reglas de entrenamiento

| Tarea | Entregable |
|-------|------------|
| Validar clasificacion ejercicios | Lista de ejercicios home_viable |
| Definir reglas de volumen | Ajustes de sets/reps para casa |
| Aprobar templates | Validacion de estructura de rutinas |
| Estrategias de gaps | Sustituciones para patrones faltantes |
| Review de prompts de IA | Asegurar coherencia fitness |

**Documentos a crear/revisar:**
- Matriz de ejercicios por equipamiento
- Reglas de sustitucion de patrones
- Guia de volumen para casa

### code-reviewer

**Responsabilidad Principal:** QA, testing y revision de codigo

| Tarea | Entregable |
|-------|------------|
| Crear test cases | Suite de E2E para CASA |
| Revisar PRs | Aprobacion de cambios |
| Integration testing | Validar flujo completo |
| Regression testing | Asegurar GYM no se rompe |
| Performance testing | Tiempos de generacion de rutinas |

**Archivos a crear:**
- `e2e/home_training_test_plan.md`
- `e2e/home_training_test_data.sql`
- Test cases en `GymRatFlow_E2E_TestRunner.json`

---

## 5. Timeline Overview

### Phase 1: Foundation (Semana 1-2)

| Actividad | Owner | Duracion |
|-----------|-------|----------|
| Finalizar spec de BD | pixel-dev | 2 dias |
| Crear migrations | pixel-dev | 2 dias |
| Clasificar ejercicios | kiro-coach | 3 dias |
| Definir templates CASA | kiro-coach | 2 dias |
| Setup ambiente testing | code-reviewer | 1 dia |

**Milestone:** Base de datos lista para CASA

### Phase 2: Workflow Development (Semana 3-4)

| Actividad | Owner | Duracion |
|-----------|-------|----------|
| Modificar KYC workflow | n8n-agent | 3 dias |
| Actualizar GymRatForm | n8n-agent | 4 dias |
| Implementar filtrado ejercicios | n8n-agent | 2 dias |
| Unit tests workflows | code-reviewer | 2 dias |

**Milestone:** Workflows funcionales en ambiente test

### Phase 3: Integration & Testing (Semana 5-6)

| Actividad | Owner | Duracion |
|-----------|-------|----------|
| Integration testing | code-reviewer | 3 dias |
| Fix bugs encontrados | n8n-agent + pixel-dev | 3 dias |
| Regression testing GYM | code-reviewer | 2 dias |
| Performance optimization | pixel-dev | 2 dias |

**Milestone:** Feature completa y testeada

### Phase 4: Launch (Semana 7)

| Actividad | Owner | Duracion |
|-----------|-------|----------|
| Deploy a produccion | pixel-dev | 1 dia |
| Monitoreo inicial | code-reviewer | 2 dias |
| Documentacion final | Todos | 1 dia |
| Retrospectiva | Todos | 1 dia |

**Milestone:** Feature live en produccion

### Gantt Overview

```
Semana:     1    2    3    4    5    6    7
Phase 1:    ████████
Phase 2:              ████████
Phase 3:                        ████████
Phase 4:                                  ████
```

---

## 6. Success Metrics

### Primary KPIs

| Metrica | Target | Medicion |
|---------|--------|----------|
| Usuarios CASA onboarded | 50+ en primer mes | Query a `users_gym_profile` |
| Tasa de completitud rutinas CASA | >70% | Query a `user_weekly_schedule` |
| Errores de generacion | <1% | Logs de n8n |
| Tiempo de generacion rutina | <30 seg | Monitoring de workflow |

### Secondary KPIs

| Metrica | Target | Medicion |
|---------|--------|----------|
| NPS usuarios CASA | >7 | Survey post-onboarding |
| Retencion 4 semanas | >60% | Analisis de cohortes |
| Conversion a GYM | >10% | Upgrade tracking |

### Quality Gates

- [ ] 100% test cases pasando
- [ ] 0 regresiones en flujo GYM
- [ ] Cobertura de codigo >80%
- [ ] Documentacion completa
- [ ] Review de seguridad aprobado

---

## 7. Risks & Mitigations

### High Risk

| Riesgo | Probabilidad | Impacto | Mitigacion |
|--------|--------------|---------|------------|
| Gaps en patrones de ejercicio | Alta | Alto | Definir sustituciones y patrones alternativos |
| Complejidad de n8n workflows | Media | Alto | Desarrollo incremental con checkpoints |
| Regresiones en flujo GYM | Media | Alto | Suite de regression testing robusta |

### Medium Risk

| Riesgo | Probabilidad | Impacto | Mitigacion |
|--------|--------------|---------|------------|
| Usuarios sin equipamiento minimo | Alta | Medio | Rutinas de peso corporal como fallback |
| Performance de queries | Media | Medio | Indices optimizados, caching |
| Prompts de IA no optimos | Media | Medio | Iteracion con kiro-coach |

### Low Risk

| Riesgo | Probabilidad | Impacto | Mitigacion |
|--------|--------------|---------|------------|
| Cambios en API de WhatsApp | Baja | Alto | Abstraccion de integracion |
| Limites de n8n | Baja | Medio | Monitoreo de uso |

### Gap Analysis - Exercise Patterns

Patrones con baja cobertura para CASA identificados:

| Patron | Ejercicios Casa | Status |
|--------|-----------------|--------|
| `pull_v` | 19 | Critico - requiere bandas/barra |
| `quads` | Bajo | Limitado sin maquinas |
| `hamstrings` | Bajo | Requiere equipamiento |
| `calfs` | Bajo | Peso corporal suficiente |

**Estrategia de mitigacion:**
1. Requerir bandas elasticas como equipamiento minimo
2. Usar variaciones de peso corporal
3. Reducir volumen de patrones con gaps
4. Sustituir con patrones similares

---

## 8. Document Index

### Planning Documents

| ID | Documento | Descripcion | Status |
|----|-----------|-------------|--------|
| 00 | `00_README.md` | Este documento - overview del proyecto | Completo |
| 01 | `01_database_spec.md` | Especificacion de cambios en BD | Pendiente |
| 02 | `02_exercise_classification.md` | Matriz de ejercicios por ambiente | Pendiente |
| 03 | `03_routine_templates.md` | Templates de rutinas para CASA | Pendiente |
| 04 | `04_workflow_changes.md` | Cambios detallados a n8n workflows | Pendiente |

### Technical Documents

| ID | Documento | Descripcion | Status |
|----|-----------|-------------|--------|
| 05 | `05_kyc_flow_spec.md` | Especificacion del nuevo flujo KYC | Pendiente |
| 06 | `06_gymratform_spec.md` | Especificacion de generacion de rutinas | Pendiente |
| 07 | `07_api_changes.md` | Cambios en workout-tracker-back (si aplica) | Pendiente |

### Testing Documents

| ID | Documento | Descripcion | Status |
|----|-----------|-------------|--------|
| 08 | `08_test_plan.md` | Plan de testing E2E | Pendiente |
| 09 | `09_test_data.sql` | Datos de prueba para CASA | Pendiente |
| 10 | `10_acceptance_criteria.md` | Criterios de aceptacion | Pendiente |

### Reference Documents

| ID | Documento | Descripcion | Status |
|----|-----------|-------------|--------|
| 11 | `11_gap_analysis.md` | Analisis detallado de gaps de ejercicios | Pendiente |
| 12 | `12_equipment_matrix.md` | Matriz de equipamiento casero | Pendiente |

---

## Appendix A: Glossary

| Termino | Definicion |
|---------|------------|
| **Environment** | Lugar de entrenamiento: GYM o CASA |
| **Home Equipment** | Equipamiento disponible en casa del usuario |
| **Gap** | Patron de movimiento sin suficientes ejercicios |
| **Template** | Estructura predefinida de rutina semanal |
| **Mesocycle** | Periodo de 4 semanas de entrenamiento |
| **Pattern** | Tipo de movimiento (push, pull, hinge, etc.) |

## Appendix B: Contact Information

| Rol | Contacto | Canal |
|-----|----------|-------|
| Product Owner | TBD | Slack |
| n8n-agent | AI Agent | Claude Code |
| pixel-dev | AI Agent | Claude Code |
| kiro-coach | AI Agent | Kiro |
| code-reviewer | AI Agent | Claude Code |

---

**Document Control**

| Version | Fecha | Autor | Cambios |
|---------|-------|-------|---------|
| 1.0 | 2026-02-03 | Claude Code | Creacion inicial |

---

*Este documento es parte del proyecto GymBot Home Training Feature. Para preguntas o sugerencias, contactar al Product Owner.*
