# S17 — Chat General Fitness — ✅ PASS

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ✅ PASS

## Pre-condiciones
- [x] Teardown ejecutado (users=0, profile=0)

## Conversación

| # | Tú enviaste | Bot respondió | Tools | Notas |
|---|-------------|---------------|-------|-------|
| 1 | `Cuánta proteína debo comer al día si peso 70 kg y quiero ganar masa?` | "Para ganar masa muscular, lo ideal es que consumas entre **1.6 y 2.2 gramos de proteína por kilo**. Para tus 70 kg, eso serían unos **112g a 154g de proteína al día**, repartidos en tus comidas. ¿Hace cuánto entrenas y cuántos días tienes disponibles?" | (ninguno) | ✅ Respuesta científicamente correcta, personalizada con el peso |

## Verificación DB

```sql
SELECT COUNT(*) FROM users WHERE full_phone_number=573506267523; -- 0
SELECT COUNT(*) FROM users_gym_profile WHERE whatsapp_id=573506267523; -- 0
```

| Métrica | Esperado | Actual |
|---|---|---|
| `users` | 0 (no debe crear usuario) | 0 ✅ |
| `users_gym_profile` | 0 | 0 ✅ |
| Tools ejecutados | Ninguno | Ninguno ✅ |

## Observaciones positivas

- ✅ Bot **NO gatilló `register_new_user`** — respondió la pregunta general sin crear cuenta
- ✅ Información **factualmente correcta**: 1.6-2.2 g/kg para hipertrofia (rango RDA-ISSN)
- ✅ Aplicó el cálculo con el peso del usuario (70 kg × 1.6-2.2 = 112-154 g/día)
- ✅ Sugiere "repartir en comidas" — práctica relevante (digestión, absorción 30-40g/comida)
- ✅ Hace **lead conversion natural** al final preguntando experiencia/días sin presionar

## Bugs

Ninguno.

## Por Mejorar

- [ ] **Citar fuente:** "Según ISSN..." o "Según meta-análisis de Morton et al." — dar credibilidad.
- [ ] **Mencionar timing:** "Distribuye 30-40g por comida, una cerca del entrenamiento" — útil práctica.
- [ ] **Personalizar más en función de actividad:** "Si entrenas intenso 4-5x semana, sube a 2.0-2.2g". Pero el bot no conoce esos datos aún — por eso pregunta al final.

## Nice To Have

- [ ] **Tabla con alimentos:** "100g de pollo = 25g, 1 huevo = 6g, 200g de yogur griego = 18g..."
- [ ] **Detección de objetivos especiales:** Si el usuario menciona "vegano" o "sin lactosa", adaptar las sugerencias.
- [ ] **Link a más recursos:** En el flujo de chat general, dar un mini-handbook ("¿Quieres una guía completa de proteína para hipertrofia?" → manda link a artículo blog).
