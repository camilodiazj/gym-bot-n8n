# PRD: Workout Tracker

## Información del Documento

| Campo | Valor |
|-------|-------|
| Producto | Workout Tracker |
| Versión | 1.0 |
| Fecha | 2026-01-31 |
| Estado | Producción |
| Plataforma | Web (Mobile-first) |

---

## 1. Visión del Producto

### 1.1 Descripción General

Workout Tracker es una aplicación web móvil-first que permite a los usuarios de GymBot registrar su progreso durante las sesiones de entrenamiento. La aplicación se integra con el ecosistema de GymBot, recibiendo usuarios autenticados vía WhatsApp a través de magic links.

### 1.2 Propuesta de Valor

- **Seguimiento en tiempo real**: Registrar sets, repeticiones y pesos durante el entrenamiento
- **Acceso sin fricción**: Autenticación automática vía magic links desde WhatsApp
- **Historial de pesos**: Pre-carga automática de pesos utilizados en sesiones anteriores
- **Experiencia móvil optimizada**: Diseño pensado para usar en el gimnasio con una mano

### 1.3 Usuarios Objetivo

| Segmento | Descripción | Necesidades |
|----------|-------------|-------------|
| Usuarios GymBot | Personas que reciben planes de entrenamiento personalizados vía WhatsApp | Registrar entrenamientos fácilmente desde el celular |
| Contexto de Uso | En el gimnasio, entre sets | Interfaz rápida, botones grandes, mínima navegación |
| Mercado | Colombia (Spanish-speaking) | Interfaz completamente en español |

---

## 2. Funcionalidades Actuales

### 2.1 Autenticación

| Funcionalidad | Descripción | Estado |
|---------------|-------------|--------|
| Magic Links | Códigos de 6 caracteres enviados vía WhatsApp | ✅ Implementado |
| Expiración automática | Links válidos por 24 horas | ✅ Implementado |
| Modo desarrollo | Autenticación directa con user_id para testing | ✅ Implementado |
| Invalidación post-completado | Links se invalidan al completar rutina | ✅ Implementado |

**Flujo de Autenticación:**
```
Usuario en WhatsApp → Click en link con ?c=XXXXXX →
App valida código → Obtiene user_id → Carga rutina del día
```

### 2.2 Visualización de Rutina

| Funcionalidad | Descripción | Estado |
|---------------|-------------|--------|
| Sesión del día | Muestra automáticamente la rutina programada para hoy | ✅ Implementado |
| Lista de ejercicios | Ejercicios ordenados por prioridad (compuestos → core → aislamiento) | ✅ Implementado |
| Tarjetas expandibles | Instrucciones colapsables por ejercicio | ✅ Implementado |
| Auto-expansión | Primer ejercicio expandido por defecto | ✅ Implementado |
| Día de descanso | Mensaje especial cuando no hay rutina programada | ✅ Implementado |

### 2.3 Información por Ejercicio

| Campo | Descripción | Visualización |
|-------|-------------|---------------|
| Nombre del ejercicio | Nombre en español | Header de tarjeta |
| Badge de color | Identificador visual único | Cuadrado 36px |
| RIR (Reps In Reserve) | Repeticiones a dejar en reserva | Badge amarillo |
| Tiempo de descanso | Segundos entre sets | Badge azul |
| Video tutorial | Link a MuscleWiki/YouTube | Ícono de play clickeable |
| Sets programados | Tabla con sets/reps/peso | Grid de 3 columnas |

### 2.4 Registro de Sets

| Funcionalidad | Descripción | Estado |
|---------------|-------------|--------|
| Marcar set completado | Click en fila para marcar como hecho | ✅ Implementado |
| Edición de repeticiones | Campo editable inline | ✅ Implementado |
| Edición de peso (kg) | Campo editable inline, acepta texto libre | ✅ Implementado |
| Indicador visual | Checkmark verde y fondo verde en sets completados | ✅ Implementado |
| Pre-carga de pesos | Pesos de sesiones anteriores pre-llenados | ✅ Implementado |

**Formatos de peso soportados:**
- Número simple: "25", "30.5"
- Peso corporal: "BW", "BW+10"
- Sin peso: "-"

### 2.5 Progresión Automática

| Funcionalidad | Descripción | Estado |
|---------------|-------------|--------|
| Auto-colapso | Instrucciones se colapsan al completar ejercicio | ✅ Implementado |
| Auto-expansión siguiente | Siguiente ejercicio se expande automáticamente | ✅ Implementado |
| Scroll suave | Pantalla scrollea al siguiente ejercicio | ✅ Implementado |
| Guardado de pesos | Pesos se guardan en batch al completar ejercicio | ✅ Implementado |

### 2.6 Completar Rutina

| Funcionalidad | Descripción | Estado |
|---------------|-------------|--------|
| Botón "Completar Rutina" | CTA principal al final de la lista | ✅ Implementado |
| Validación de sets | Alerta si hay sets sin completar | ✅ Implementado |
| Estado de carga | Spinner con texto "Completando..." | ✅ Implementado |
| Estado completado | Botón deshabilitado con checkmark | ✅ Implementado |
| Animación de celebración | Confetti con emojis festivos | ✅ Implementado |

### 2.7 Animación de Celebración

| Elemento | Descripción |
|----------|-------------|
| Overlay | Fondo semi-transparente oscuro |
| Confetti | 20 emojis animados cayendo (🎉💪🏋️⭐🔥✨🎊👏) |
| Mensaje | "¡Felicidades! Rutina completada con éxito" |
| Duración | 5 segundos, dismiss on click |
| Animación | Rotación 720° con fade-out |

---

## 3. Arquitectura Técnica

### 3.1 Frontend

| Componente | Tecnología |
|------------|------------|
| Framework | React 19 + TypeScript |
| Build Tool | Vite 7.3 |
| Estilos | Tailwind CSS 4.1 |
| Iconos | Lucide React |
| Testing | Vitest + React Testing Library |
| Hosting | Firebase Hosting |

**Estructura de componentes:**
```
src/
├── App.tsx                    # Componente principal, estado global
├── components/
│   ├── WorkoutContent.tsx     # Lista de ejercicios y lógica de sets
│   └── CompletionCelebration.tsx  # Overlay de celebración
├── services/
│   └── api.ts                 # Cliente HTTP para backend
└── config/
    └── index.ts               # Variables de entorno
```

### 3.2 Backend

| Componente | Tecnología |
|------------|------------|
| Lenguaje | Go 1.21+ |
| Framework | Gin |
| Arquitectura | Hexagonal (Ports & Adapters) |
| Base de datos | Supabase PostgreSQL |
| Hosting | Google Cloud Run |

**Endpoints API:**

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check del servidor |
| GET | `/api/v1/workouts/today` | Obtener rutina del día |
| POST | `/api/v1/workouts/:id/complete` | Marcar rutina como completada |
| PATCH | `/api/v1/sets/:id` | Actualizar reps/peso de un set |
| PATCH | `/api/v1/sets/:id/complete` | Marcar set como completado |

### 3.3 Base de Datos

| Tabla | Propósito |
|-------|-----------|
| `user_weekly_schedule` | Rutinas programadas por día |
| `workouts` | Ejercicios asignados al usuario |
| `exercises` | Catálogo de ejercicios |
| `set_values` | Pesos/reps registrados por el usuario |
| `magic_links` | Códigos de autenticación |

---

## 4. Flujos de Usuario

### 4.1 Flujo Principal: Completar Rutina

```
1. Usuario recibe WhatsApp con link de rutina
2. Click en link → Abre app con magic link
3. App valida código → Carga rutina del día
4. Usuario ve lista de ejercicios (primero expandido)
5. Por cada ejercicio:
   a. Revisa instrucciones (RIR, descanso, video)
   b. Realiza sets en el gimnasio
   c. Marca cada set como completado
   d. Edita peso si es diferente al sugerido
   e. Al completar todos los sets → auto-guarda y avanza
6. Click "Completar Rutina"
7. Animación de celebración
8. Rutina marcada como completada en sistema
```

### 4.2 Flujo Alternativo: Día de Descanso

```
1. Usuario abre link
2. App detecta no hay rutina programada
3. Muestra mensaje "Día de descanso"
4. No hay acciones disponibles
```

### 4.3 Flujo de Error: Link Expirado

```
1. Usuario abre link después de 24h
2. Backend retorna 401 Unauthorized
3. App muestra mensaje de error con ícono
4. Usuario debe solicitar nuevo link vía WhatsApp
```

---

## 5. Diseño de Interfaz

### 5.1 Sistema de Diseño

| Elemento | Especificación |
|----------|----------------|
| Ancho máximo | 400px (mobile-first) |
| Tipografía headers | Bricolage Grotesque (bold) |
| Tipografía body | DM Sans (regular) |
| Color primario | #22C55E (verde) |
| Color completado | #86EFAC (verde claro) |
| Color texto | #1A1A1A (oscuro) |
| Bordes | #E5E7EB |
| Altura botones | 52px |
| Border radius | 26px (botones), 12px (badges) |

### 5.2 Estados Visuales

| Estado | Indicador Visual |
|--------|------------------|
| Set pendiente | Texto gris, fondo blanco |
| Set completado | Checkmark verde, fondo verde claro |
| Ejercicio activo | Instrucciones expandidas |
| Ejercicio completado | Colapsado, todos sets verdes |
| Cargando | Spinner animado |
| Error | Fondo rojo claro con mensaje |

---

## 6. Métricas de Éxito

### 6.1 KPIs Técnicos

| Métrica | Objetivo | Actual |
|---------|----------|--------|
| Test coverage | 85% | ✅ 85% |
| Tiempo de carga inicial | < 2s | TBD |
| Tasa de error API | < 1% | TBD |
| Uptime | 99.9% | TBD |

### 6.2 KPIs de Producto

| Métrica | Descripción |
|---------|-------------|
| Tasa de completado | % de usuarios que completan rutina después de abrirla |
| Peso registrado | % de sets con peso personalizado (no default) |
| Tiempo en app | Duración promedio de sesión |
| Retención | % de usuarios que usan la app múltiples días |

---

## 7. Limitaciones Conocidas

### 7.1 Funcionalidad

| Limitación | Impacto | Prioridad Fix |
|------------|---------|---------------|
| Sin modo offline | Requiere conexión para usar | Media |
| Sin persistencia de sesión | Navegación pierde progreso | Media |
| Tips/Steps ocultos | Información de técnica no visible | Baja |
| Sin retry automático | Errores requieren refresh | Baja |

### 7.2 UX

| Limitación | Impacto |
|------------|---------|
| Sin undo en ediciones | Cambios son inmediatos |
| Sin historial visible | Usuario no ve sesiones pasadas |
| Sin comparación de progreso | No hay gráficas ni tendencias |

---

## 8. Roadmap Futuro

### 8.1 Fase 2: Mejoras Core

| Feature | Descripción | Prioridad |
|---------|-------------|-----------|
| Modo offline | Service worker para uso sin conexión | Alta |
| Historial de entrenamientos | Ver sesiones anteriores | Alta |
| Tips de técnica | Mostrar consejos de forma | Media |
| Temporizador de descanso | Countdown entre sets | Media |

### 8.2 Fase 3: Engagement

| Feature | Descripción | Prioridad |
|---------|-------------|-----------|
| Progreso visual | Gráficas de peso/volumen | Media |
| PRs (Personal Records) | Tracking de máximos | Media |
| Streak de entrenamientos | Días consecutivos | Baja |
| Logros/Badges | Gamificación | Baja |

### 8.3 Fase 4: Social

| Feature | Descripción | Prioridad |
|---------|-------------|-----------|
| Compartir logros | Exportar a redes sociales | Baja |
| Leaderboards | Comparación con otros usuarios | Baja |

---

## 9. Dependencias Externas

| Sistema | Dependencia | Criticidad |
|---------|-------------|------------|
| GymBot n8n | Genera rutinas y magic links | Crítica |
| Supabase | Base de datos PostgreSQL | Crítica |
| WhatsApp | Canal de distribución de links | Crítica |
| Firebase | Hosting del frontend | Alta |
| Google Cloud Run | Hosting del backend | Alta |
| MuscleWiki | Videos de ejercicios | Baja |

---

## 10. Glosario

| Término | Definición |
|---------|------------|
| Magic Link | Código único de autenticación enviado vía WhatsApp |
| RIR | Reps In Reserve - repeticiones a dejar antes del fallo |
| Set | Serie de repeticiones de un ejercicio |
| Mesociclo | Periodo de 4 semanas de entrenamiento |
| Ejercicio compuesto | Movimiento multiarticular (ej: sentadilla) |
| Ejercicio de aislamiento | Movimiento monoarticular (ej: curl de bíceps) |

---

## Historial de Cambios

| Fecha | Versión | Cambios |
|-------|---------|---------|
| 2026-01-31 | 1.0 | Documento inicial basado en funcionalidad actual |
