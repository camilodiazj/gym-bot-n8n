# Feature Specification: Agente Unificado Kairos

**Feature Branch**: `001-kairos-unified-agent`
**Created**: 2026-03-17
**Status**: Draft

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Usuario Activo Ve y Confirma Rutina (Priority: P1)

Un usuario existente con plan activo puede preguntarle a Kairos qué le toca entrenar hoy, recibir su rutina formateada, y confirmar que la completó — todo en la misma conversación y sin necesidad de comandos especiales.

**Why this priority**: Es el flujo diario principal. La mayoría de interacciones son de usuarios que ya tienen rutina y quieren verla o reportar que entrenaron. Sin esto, el sistema no tiene valor operativo.

**Independent Test**: Puede probarse completamente enviando mensajes como usuario con plan activo y verificando que la rutina se muestra correcta y la sesión queda marcada como completada.

**Acceptance Scenarios**:

1. **Given** un usuario con sesión programada hoy sin completar, **When** pregunta "¿qué me toca hoy?", **Then** el agente responde con la lista de ejercicios, series, reps, descanso y RIR de la sesión correspondiente.
2. **Given** un usuario con sesión mostrada anteriormente, **When** dice "ya terminé mi rutina", **Then** el agente confirma la sesión como completada y responde con mensaje motivacional.
3. **Given** un usuario sin sesión hoy pero con sesión perdida en los últimos 3 días, **When** dice "quiero entrenar hoy", **Then** el agente ofrece proactivamente la sesión pendiente.
4. **Given** un usuario sin sesión hoy y sin sesiones pendientes, **When** dice "quiero entrenar hoy", **Then** el agente informa el día de descanso y menciona la próxima sesión programada.

---

### User Story 2 - Usuario Nuevo Completa Onboarding KYC (Priority: P1)

Una persona que escribe por primera vez es guiada automáticamente por un flujo de recopilación de datos (nombre, objetivo, experiencia, días disponibles, ambiente, datos físicos, salud) y al final su perfil queda guardado listo para crear una rutina.

**Why this priority**: Sin el KYC no hay perfil y sin perfil no hay rutina. Es la puerta de entrada al sistema y debe funcionar impecablemente para no perder usuarios en el primer contacto.

**Independent Test**: Puede probarse con un número de teléfono nuevo enviando "Hola" y completando todos los turnos del KYC hasta que el perfil quede guardado en la base de datos.

**Acceptance Scenarios**:

1. **Given** un número de teléfono que no existe en el sistema, **When** envía cualquier mensaje, **Then** el agente inicia el flujo KYC con el primer campo de recopilación.
2. **Given** un usuario en medio del KYC, **When** proporciona información con errores (ej. edad inválida), **Then** el agente solicita corrección sin perder los datos ya recopilados.
3. **Given** un usuario que completó todos los campos del KYC, **When** confirma que el perfil es correcto, **Then** el perfil queda guardado y el agente ofrece crear la rutina.
4. **Given** un usuario que ya completó KYC en una sesión anterior, **When** escribe en una nueva sesión, **Then** el agente NO repite el KYC y responde como usuario existente.

---

### User Story 3 - Creación de Rutina en Modo Borrador (Priority: P2)

Después del KYC, el agente ofrece crear la rutina de forma interactiva: pregunta al usuario cómo prefiere el proceso (todo junto o día por día), genera un borrador, permite cambios de ejercicios según feedback del usuario, y guarda la versión final aprobada.

**Why this priority**: Diferenciador clave frente al proceso actual. El usuario puede opinar sobre su rutina antes de que quede guardada, reduciendo insatisfacción post-generación.

**Independent Test**: Puede probarse completamente iniciando el flujo de creación con un usuario que tiene perfil pero sin plan, haciendo al menos un cambio de ejercicio, y verificando que el plan final guardado refleja las modificaciones solicitadas.

**Acceptance Scenarios**:

1. **Given** un usuario con perfil completo pero sin plan, **When** solicita su rutina, **Then** el agente pregunta si prefiere ver todo el borrador junto o día por día.
2. **Given** el agente generó un borrador completo, **When** el usuario dice "no me gusta [ejercicio X]", **Then** el agente busca una alternativa compatible con el perfil y la propone.
3. **Given** el agente propuso una alternativa, **When** el usuario aprueba, **Then** el agente actualiza el borrador mostrando el cambio aplicado.
4. **Given** el usuario aprobó el borrador final, **When** confirma que quiere guardarlo, **Then** el plan queda persistido con 4 semanas de workouts y el agente ofrece agendar los días de entrenamiento.
5. **Given** el usuario tiene una rutina previa y dice "hazla igual pero cambia [ejercicio]", **Then** el agente aplica el swap directamente sin mostrar borrador completo.

---

### User Story 4 - Agendamiento de Sesiones (Priority: P2)

Un usuario con plan activo pero sin días programados puede decirle al agente qué días quiere entrenar y el sistema crea las sesiones en el calendario.

**Why this priority**: Sin schedule el sistema de recordatorios y seguimiento no funciona. Es el paso natural después de crear la rutina.

**Independent Test**: Puede probarse con un usuario que tiene plan pero sin sesiones en su calendario, diciéndole al agente qué días quiere entrenar y verificando que las sesiones quedan creadas correctamente.

**Acceptance Scenarios**:

1. **Given** un usuario sin sesiones agendadas, **When** dice "quiero programar mis entrenamientos", **Then** el agente le informa cuántas sesiones tiene su plan y pregunta qué días prefiere.
2. **Given** el agente preguntó los días, **When** el usuario responde con la cantidad correcta de días, **Then** el agente crea las sesiones y confirma el horario resultante.
3. **Given** el usuario da más o menos días de los que requiere el plan, **Then** el agente informa la discrepancia y solicita aclaración.

---

### User Story 5 - Tareas Pendientes y Chat de Fitness (Priority: P3)

Cuando el usuario tiene una tarea pendiente (confirmar si entrenó ayer), el agente la resuelve primero antes de atender cualquier otra solicitud. Además, el agente responde preguntas generales de fitness personalizadas al perfil del usuario sin herramientas externas.

**Why this priority**: El manejo de tareas pendientes completa el flujo de seguimiento nocturno. El chat de fitness aumenta retención pero no es bloqueante para el valor principal.

**Independent Test**: Puede probarse enviando un saludo con una tarea pendiente activa y verificando que el agente pregunta por la tarea antes de responder cualquier otra cosa.

**Acceptance Scenarios**:

1. **Given** un usuario con tarea pendiente de confirmar rutina, **When** envía cualquier mensaje, **Then** el agente pregunta primero si completó la sesión pendiente.
2. **Given** el agente preguntó por la tarea y el usuario responde "sí la hice", **Then** la sesión queda marcada como completada y la tarea como resuelta.
3. **Given** el agente preguntó por la tarea y el usuario responde "no pude ir", **Then** la tarea queda marcada como declinada y el agente continúa con el flujo normal.
4. **Given** un usuario con plan activo hace una pregunta general de nutrición o fitness, **Then** el agente responde personalizando con el objetivo y nivel del usuario, sin inventar datos de rutina.

---

### User Story 6 - Renovación de Mesociclo (Priority: P3)

Cuando un usuario completa todas las sesiones de la semana 4, el agente detecta automáticamente que el mesociclo está listo para renovarse y ofrece opciones: mantener la misma rutina con progresión o cambiar ejercicios.

**Why this priority**: Necesario para la continuidad del programa más allá de las 4 semanas iniciales, pero no bloquea el valor inmediato del sistema.

**Independent Test**: Puede probarse con un usuario que tiene todas las sesiones de semana 4 completadas, enviando "hola" y verificando que el agente ofrece las opciones de renovación.

**Acceptance Scenarios**:

1. **Given** un usuario con todas las sesiones de semana 4 completadas, **When** envía cualquier mensaje, **Then** el agente detecta el mesociclo completado y ofrece opciones de renovación.
2. **Given** el agente ofreció opciones de renovación, **When** el usuario elige mantener la rutina, **Then** el agente confirma la renovación con progresión de carga para el nuevo ciclo.
3. **Given** el usuario pide mantener pero con más días de entrenamiento, **Then** el agente informa que el esquema cambiará y solicita confirmación explícita.

---

### Edge Cases

- ¿Qué pasa si el usuario envía un mensaje de estado de WhatsApp (no es un mensaje real)?
- ¿Qué pasa si la base de datos no responde al cargar el contexto?
- ¿Qué pasa si el usuario confirma entrenamiento pero no tenía sesión programada hoy ni ayer?
- ¿Qué pasa si el usuario solicita cambiar un ejercicio del borrador a uno que no existe en el catálogo?
- ¿Qué pasa si el usuario está en medio del KYC y envía un mensaje completamente fuera de contexto?
- ¿Cómo maneja el agente múltiples intenciones en un solo mensaje (ej. "dame la rutina y también el link")?
- ¿Qué pasa si el usuario tiene sesión hoy Y sesiones perdidas — cuál prioriza el agente?

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema DEBE cargar el contexto completo del usuario (perfil, plan activo, sesiones de hoy, sesiones perdidas de los últimos 3 días, tareas pendientes) desde la base de datos antes de cada turno del agente.
- **FR-002**: El sistema DEBE enrutar automáticamente a usuarios nuevos sin perfil al flujo de onboarding KYC, y a usuarios existentes al agente conversacional.
- **FR-003**: El agente DEBE resolver tareas pendientes de tipo confirmación de rutina antes de atender cualquier otra solicitud del usuario en el mismo turno.
- **FR-004**: El agente DEBE ofrecer proactivamente sesiones perdidas (hasta 3 días atrás sin completar) cuando un usuario sin sesión hoy solicita entrenar.
- **FR-005**: El agente DEBE generar borradores de rutina consultando el catálogo de ejercicios, plantillas de días y parámetros de carga — nunca inventando datos.
- **FR-006**: El agente DEBE permitir al usuario modificar ejercicios del borrador antes de guardarlo, buscando alternativas compatibles con el perfil del usuario.
- **FR-007**: El sistema DEBE persistir el plan de entrenamiento (plan + 4 semanas de workouts) únicamente cuando el usuario aprueba explícitamente el borrador final.
- **FR-008**: El agente DEBE generar enlaces de acceso al Workout Tracker web con expiración de 48 horas cuando el usuario lo solicite.
- **FR-009**: El agente DEBE detectar automáticamente cuando un usuario completó todas las sesiones de semana 4 y ofrecer renovación de mesociclo.
- **FR-010**: El agente DEBE responder preguntas generales de fitness personalizando la respuesta con el objetivo, nivel y datos del perfil del usuario, sin consultar herramientas externas.
- **FR-011**: El sistema DEBE mantener memoria de conversación por número de teléfono de forma que cada usuario tenga un hilo independiente y aislado.
- **FR-012**: El agente DEBE validar que la cantidad de días que el usuario indica para agendar coincide con el número de sesiones del plan activo antes de crearlas.
- **FR-013**: El sistema DEBE ignorar mensajes que no son interacciones de usuario real (ej. actualizaciones de estado de WhatsApp).

### Key Entities

- **UserContext**: Snapshot del estado del usuario cargado antes de cada turno. Incluye identidad, plan activo, sesiones de hoy, sesiones perdidas, próxima sesión, tareas pendientes y flags de estado (nuevo usuario, KYC completo, tiene schedule, semana 4 completada).
- **DraftRoutine**: Borrador de rutina en construcción durante la creación interactiva. Contiene los días con ejercicios seleccionados antes de ser persistido. No existe en almacenamiento permanente hasta que el usuario aprueba.
- **Session**: Sesión de entrenamiento programada para un día específico. Puede estar en estado: completada, pendiente (hoy o futuro), o perdida (pasado sin completar).
- **PendingTask**: Tarea que el agente debe resolver antes de continuar el flujo normal. El tipo principal es confirmación de rutina del día anterior.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los mensajes de usuarios existentes reciben una respuesta que usa datos reales de su perfil, no respuestas genéricas.
- **SC-002**: Los usuarios pueden ver su rutina del día, confirmar entrenamiento y obtener el link del tracker en una sola conversación de 3 mensajes o menos.
- **SC-003**: El flujo completo de creación de rutina con borrador (incluyendo al menos un cambio de ejercicio) se completa en 5 mensajes o menos.
- **SC-004**: El agente detecta y ofrece sesiones perdidas en el 100% de los casos donde existen sesiones sin completar en los últimos 3 días.
- **SC-005**: El agente resuelve tareas pendientes antes de continuar en el 100% de los casos donde existen tareas activas.
- **SC-006**: Los usuarios completan el flujo KYC con tasa de abandono menor al 20%.
- **SC-007**: Ningún usuario puede acceder a datos de otro usuario — aislamiento completo por número de teléfono.
- **SC-008**: El agente responde sin consultar herramientas externas en el 100% de las preguntas generales de fitness.

---

## Assumptions

- Los usuarios interactúan exclusivamente por WhatsApp (canal de texto, mensajes cortos de máximo 3-4 oraciones en las respuestas del agente).
- El contexto del usuario se re-carga en cada turno de conversación — no se asume que el contexto del turno anterior sigue vigente.
- El catálogo de ejercicios y las plantillas de rutina ya existen correctamente en la base de datos.
- El flujo KYC de Case 5 funciona correctamente y no requiere modificaciones para ser reutilizado.
- La generación de rutina en borrador cubre solo el mesociclo 1 para usuarios nuevos. Los mesociclos posteriores se manejan por renovación.
- Los parámetros de carga (series, reps, RIR, descanso) se determinan automáticamente basados en objetivo y nivel — no se negocian con el usuario durante el borrador.

---

## Dependencies

- **Case 5 (KYC Subgraph)**: El flujo KYC existente se reutiliza como subgrafo sin modificaciones.
- **Base de datos (Supabase)**: Todas las herramientas del agente dependen de consultas a Supabase. Su disponibilidad es crítica para cada turno.
- **Workout Tracker (frontend web)**: La generación de links de acceso requiere que el frontend esté desplegado y operativo.
- **Catálogo de ejercicios**: La creación de borradores de rutina requiere datos correctamente clasificados en el catálogo.
