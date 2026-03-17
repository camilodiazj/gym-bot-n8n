# Case 6 — Agente Unificado Kairos: Casos de Uso

## Caso 1: Usuario Nuevo (KYC + Primera Rutina)

> Sofia nunca ha usado GymBot. Escribe por WhatsApp por primera vez.

```mermaid
sequenceDiagram
    participant U as Sofia (WhatsApp)
    participant LC as load_context
    participant R as router
    participant KYC as KYC Subgraph (Case 5)
    participant A as kairos_agent
    participant T as Tools (Supabase)
    participant DB as Supabase

    Note over U,DB: Sofia escribe por primera vez

    U->>LC: "Hola! Quiero empezar a entrenar"
    LC->>DB: SELECT users WHERE phone = '573001112222'
    DB-->>LC: [] (no existe)
    LC->>DB: SELECT users_gym_profile WHERE whatsapp_id = 573001112222
    DB-->>LC: [] (no existe)
    LC-->>R: UserContext { is_new_user: true, kyc_complete: false }

    R->>KYC: is_new_user && !kyc_complete → KYC flow

    Note over KYC,U: 5 turnos de KYC (Case 5 intacto)
    KYC-->>U: "Hola Sofia! Cual es tu objetivo?"
    U->>KYC: "Ganar masa muscular"
    KYC-->>U: "Experiencia, dias, horario?"
    U->>KYC: "3 anios, 4 dias, mananas"
    KYC-->>U: "Gym o casa?"
    U->>KYC: "Gym"
    KYC-->>U: "Datos fisicos?"
    U->>KYC: "Mujer, 25, 165cm, 58kg"
    KYC-->>U: "Lesiones?"
    U->>KYC: "Ninguna"
    KYC-->>U: "Perfil completo! Todo correcto?"
    U->>KYC: "Si, dale"
    KYC->>DB: INSERT users + users_gym_profile
    KYC-->>U: "Perfil guardado! Voy a crear tu rutina"

    Note over U,DB: Siguiente mensaje: Sofia ya existe en DB

    U->>LC: "Listo, y ahora que?"
    LC->>DB: SELECT users WHERE phone = '573001112222'
    DB-->>LC: { user_id: '...', full_name: 'Sofia' }
    LC->>DB: SELECT users_gym_profile, users_plans, user_weekly_schedule
    DB-->>LC: profile existe, plan: null, schedule: []
    LC-->>R: UserContext { is_new_user: false, kyc_complete: true, plan: null, has_schedule: false }

    R->>A: kyc_complete → Agent mode
    Note over A: Contexto: tiene perfil pero NO tiene plan ni schedule
    A-->>U: "Sofia, tu perfil esta listo pero aun no tienes<br/>rutina generada. Estamos preparandola!"
```

---

## Caso 2: Usuario Activo (Multiples Interacciones en un Dia)

> Camilo tiene plan activo, rutina de hoy "Upper Body A" sin completar.
> Interactua 3 veces en el dia: ve rutina, pide link del tracker, confirma.

```mermaid
sequenceDiagram
    participant U as Camilo (WhatsApp)
    participant LC as load_context
    participant R as router
    participant A as kairos_agent (Gemini)
    participant TN as ToolNode
    participant DB as Supabase

    Note over U,DB: 7:00 AM - Camilo quiere ver su rutina

    U->>LC: "Buenos dias, que me toca hoy?"
    LC->>DB: SELECT users, users_plans, user_weekly_schedule, pending_tasks
    DB-->>LC: user existe, plan activo, hoy: "Upper Body A" (W2, !Completed)
    LC-->>R: UserContext { is_new_user: false, kyc_complete: true,<br/>plan: {goal: 'Ganar masa', level: 'Intermedio', week: 2},<br/>todays_sessions: [{session_name: 'Upper Body A', Completed: false}],<br/>pending_tasks: [] }

    R->>A: Usuario existente → Agent mode
    Note over A: System prompt incluye:<br/>"Horario hoy: Upper Body A (NO completada)"<br/>+ usuario pide ver rutina<br/>→ Decide: llamar get_todays_routine

    A->>TN: tool_call: get_todays_routine(user_id, "Upper Body A", 2)
    TN->>DB: SELECT workouts JOIN exercises<br/>WHERE user_id AND week=2 AND day_name='Upper Body A'<br/>ORDER BY exercise_order
    DB-->>TN: [{Press banca: 4x8-10 RIR 2}, {Remo con barra: 4x8-10},<br/>{Press militar: 3x10-12}, {Curl biceps: 3x12-15}, ...]
    TN-->>A: Rutina formateada JSON
    Note over A: No mas tool calls → responde
    A-->>U: "Hoy te toca Upper Body A (Semana 2):<br/>1. Press banca 4x8-10 RIR 2 (150s)<br/>2. Remo con barra 4x8-10 RIR 2<br/>3. Press militar 3x10-12 RIR 1<br/>4. Curl biceps 3x12-15<br/>Dale con todo!"

    Note over U,DB: 7:30 AM - Camilo quiere registrar pesos en la web

    U->>LC: "Pasame el link para registrar mis pesos"
    LC->>DB: (mismas queries de contexto)
    DB-->>LC: (mismo contexto, Upper Body A sin completar)
    LC-->>R: UserContext { ... same ... }

    R->>A: Agent mode
    Note over A: Usuario pide link del tracker<br/>→ Decide: llamar create_magic_link

    A->>TN: tool_call: create_magic_link(user_id)
    TN->>DB: INSERT INTO magic_links (code, user_id, expires_at)<br/>VALUES ('a3f7b2', '...', NOW() + 48h)
    DB-->>TN: { code: 'a3f7b2' }
    TN-->>A: "https://gymbot.web.app/w?c=a3f7b2"
    A-->>U: "Aqui tienes tu link para el Workout Tracker:<br/>https://gymbot.web.app/w?c=a3f7b2<br/>Valido por 48 horas!"

    Note over U,DB: 8:30 PM - Camilo termino, confirma

    U->>LC: "Ya termine mi rutina de hoy"
    LC->>DB: (queries de contexto)
    DB-->>LC: Upper Body A aun sin completar
    LC-->>R: UserContext { todays_sessions: [{Completed: false}] }

    R->>A: Agent mode
    Note over A: Usuario dice que termino<br/>→ Decide: llamar confirm_workout_completion

    A->>TN: tool_call: confirm_workout_completion(user_id)
    TN->>DB: UPDATE user_weekly_schedule<br/>SET "Completed" = true<br/>WHERE user_id AND planned_day = TODAY
    TN->>DB: UPDATE pending_tasks<br/>SET status = 'completed'<br/>WHERE user_id AND task_type = 'CONFIRMAR_RUTINA'
    DB-->>TN: OK
    TN-->>A: "Workout marcado como completado"
    A-->>U: "Excelente Camilo! Rutina completada.<br/>Gran trabajo hoy con el Upper Body.<br/>Descansa bien, manana te espera Leg Day!"
```

---

## Caso 3: Usuario con Tarea Pendiente + Chat Libre + Mesociclo

> Ana tiene una tarea pendiente de ayer (no confirmo si entreno).
> Luego hace preguntas de nutricion. Despues consulta su mesociclo.
> El agente prioriza la tarea pendiente, responde chat sin tools, y usa tools para mesociclo.

```mermaid
sequenceDiagram
    participant U as Ana (WhatsApp)
    participant LC as load_context
    participant R as router
    participant A as kairos_agent (Gemini)
    participant TN as ToolNode
    participant DB as Supabase

    Note over U,DB: Ana escribe "hola" pero tiene tarea pendiente

    U->>LC: "Hola Kairos"
    LC->>DB: SELECT users, plans, schedule, pending_tasks
    DB-->>LC: user existe, plan activo (W4),<br/>pending_task: {task_type: 'CONFIRMAR_RUTINA',<br/>session_name: 'Full Body B', status: 'pending'}
    LC-->>R: UserContext { pending_tasks: [{CONFIRMAR_RUTINA, 'Full Body B'}],<br/>plan: {mesocycle_number: 1, week: 4} }

    R->>A: Agent mode
    Note over A: System prompt incluye:<br/>"Tareas pendientes: Confirmar Full Body B (ayer)"<br/>REGLA: Si hay tarea pendiente, preguntar primero<br/>→ Decide: NO llamar tools, preguntar directamente

    A-->>U: "Hola Ana! Vi que ayer tenias Full Body B<br/>programado. Lo completaste?"

    Note over U,DB: Ana confirma que si entreno

    U->>LC: "Si, lo hice completo"
    LC->>DB: (queries de contexto — pending_task sigue pendiente)
    LC-->>R: UserContext { pending_tasks: [{CONFIRMAR_RUTINA}] }

    R->>A: Agent mode
    Note over A: Usuario confirma que si lo hizo<br/>→ Decide: llamar confirm_workout_completion

    A->>TN: tool_call: confirm_workout_completion(user_id)
    TN->>DB: UPDATE user_weekly_schedule SET Completed = true<br/>WHERE session = 'Full Body B' AND planned_day = YESTERDAY
    TN->>DB: UPDATE pending_tasks SET status = 'completed'
    DB-->>TN: OK
    TN-->>A: "Completado"
    A-->>U: "Genial Ana! Full Body B marcado como completado.<br/>Ya llevas 3/4 sesiones esta semana!"

    Note over U,DB: Ana hace pregunta de nutricion (NO requiere tools)

    U->>LC: "Oye, cuanta proteina debo comer al dia?"
    LC->>DB: (queries de contexto)
    DB-->>LC: pending_tasks: [], plan: {goal: 'Bajar grasa', level: 'Principiante'}
    LC-->>R: UserContext { pending_tasks: [], plan: {goal: 'Bajar grasa'} }

    R->>A: Agent mode
    Note over A: Pregunta general de nutricion<br/>Contexto: goal = "Bajar grasa", peso 65kg<br/>→ Decide: NO llamar tools, responder directo

    A-->>U: "Para tu objetivo de bajar grasa, te recomiendo<br/>1.6-2.0g de proteina por kg de peso.<br/>Eso seria entre 104-130g al dia.<br/>Pollo, pescado, huevos y legumbres son tus aliados!"

    Note over U,DB: Ana pregunta por su mesociclo (semana 4)

    U->>LC: "Y que pasa cuando termine esta semana de rutina?"
    LC->>DB: (queries)
    DB-->>LC: plan: {week: 4, mesocycle_number: 1}
    LC-->>R: UserContext { plan: {week: 4, mesocycle_number: 1} }

    R->>A: Agent mode
    Note over A: Contexto: semana 4 (ultima del mesociclo)<br/>Usuario pregunta que sigue<br/>→ Decide: llamar get_mesocycle_status

    A->>TN: tool_call: get_mesocycle_status(user_id)
    TN->>DB: SELECT users_plans, user_weekly_schedule<br/>WHERE week = 4 AND Completed = true
    DB-->>TN: { week4_completed: 2, week4_total: 3,<br/>mesocycle: 1, can_renew: true }
    TN-->>A: "Semana 4, 2/3 completadas, puede renovar"
    A-->>U: "Estas en la semana 4, la ultima de tu mesociclo!<br/>Cuando completes esta semana, podemos:<br/>1. Mantener tu rutina actual (con progresion)<br/>2. Cambiar ejercicios para mas variedad<br/>Te aviso cuando sea momento de renovar!"
```

---

## Caso 4: Sesion Perdida — "Quiero entrenar hoy" (sin sesion agendada)

> Camilo entrena Lunes, Miercoles y Viernes. El Lunes no pudo ir.
> El Martes (dia de descanso) le dice al agente que quiere entrenar.
> El agente ve la sesion pendiente del Lunes y la ofrece.

```mermaid
sequenceDiagram
    participant U as Camilo (WhatsApp)
    participant LC as load_context
    participant R as router
    participant A as kairos_agent (Gemini)
    participant TN as ToolNode
    participant DB as Supabase

    Note over U,DB: Martes — Camilo NO tenia sesion agendada hoy.<br/>Pero el Lunes no entreno (Upper Body A sin completar)

    U->>LC: "Quiero entrenar hoy, dame mi rutina"
    LC->>DB: SELECT user_weekly_schedule<br/>WHERE user_id AND planned_day = TODAY
    DB-->>LC: [] (nada para Martes)
    LC->>DB: SELECT user_weekly_schedule<br/>WHERE user_id AND Completed = false<br/>AND planned_day >= TODAY - 3 days
    DB-->>LC: [{session_name: 'Upper Body A', week: 2,<br/>planned_day: '2026-03-16' (Lunes)}]
    LC-->>R: UserContext {<br/>  todays_sessions: [],<br/>  missed_sessions: [{session_name: 'Upper Body A',<br/>    week: 2, planned_day: '2026-03-16'}],<br/>  next_scheduled_session: {session_name: 'Lower Body A',<br/>    planned_day: '2026-03-18' (Miercoles)}<br/>}

    R->>A: Agent mode
    Note over A: System prompt incluye:<br/>"Horario hoy: Ninguno (dia de descanso)"<br/>"Sesiones pendientes: Upper Body A (Lunes 16/03)"<br/>+ usuario dice "quiero entrenar hoy"<br/>→ Decide: ofrecer la sesion pendiente

    A-->>U: "Hoy no tienes sesion agendada, pero tienes<br/>pendiente tu Upper Body A del lunes.<br/>Quieres hacerla hoy?"

    U->>LC: "Si, dale"
    LC->>DB: (mismas queries)
    LC-->>R: UserContext { missed_sessions: [Upper Body A] }

    R->>A: Agent mode
    Note over A: Confirmo que quiere la pendiente<br/>→ Decide: llamar get_todays_routine

    A->>TN: tool_call: get_todays_routine(user_id, "Upper Body A", 2)
    TN->>DB: SELECT workouts JOIN exercises<br/>WHERE user_id AND week=2 AND day_name='Upper Body A'
    DB-->>TN: [{Press banca: 4x8-10}, {Remo: 4x8-10}, ...]
    TN-->>A: Rutina formateada
    A-->>U: "Upper Body A — Semana 2:<br/>1. Press banca 4x8-10 RIR 2 (150s)<br/>2. Remo con barra 4x8-10 RIR 2<br/>3. Press militar 3x10-12 RIR 1<br/>Dale con todo!"
```

### Variante: Sin sesion pendiente

```mermaid
sequenceDiagram
    participant U as Camilo (WhatsApp)
    participant LC as load_context
    participant A as kairos_agent

    U->>LC: "Quiero entrenar hoy"
    Note over LC: todays_sessions: []<br/>missed_sessions: []<br/>next_scheduled: Lower Body A (Miercoles)

    LC-->>A: Agent mode
    Note over A: No hay sesion hoy NI pendientes<br/>→ Decide: informar dia de descanso + proxima sesion

    A-->>U: "Hoy es tu dia de descanso, Camilo.<br/>Tu proxima sesion es Lower Body A el miercoles.<br/>Quieres que te la muestre de una vez?"
```

**Por que es mejor que n8n:** n8n diria "No tienes rutina hoy" y punto. El agente ve la sesion pendiente y la ofrece proactivamente.

---

## Caso 5: Chat Libre — Preguntas de Fitness (sin tools)

> Camilo ya completo su rutina del dia. Hace preguntas generales.
> El agente responde usando su conocimiento + contexto del usuario, SIN llamar tools.

```mermaid
sequenceDiagram
    participant U as Camilo (WhatsApp)
    participant LC as load_context
    participant A as kairos_agent (Gemini)

    Note over U,A: Camilo ya completo Upper Body A hoy

    U->>LC: "Cuanta proteina debo comer al dia?"
    LC-->>A: UserContext { plan: {goal: 'Ganar masa muscular'},<br/>todays_sessions: [{Completed: true}],<br/>pending_tasks: [] }

    Note over A: Pregunta de nutricion general<br/>Contexto: goal = "Ganar masa muscular", peso 67kg<br/>→ Decide: NO llamar tools, responder directo

    A-->>U: "Para ganar masa muscular, lo ideal es<br/>1.6-2.2g de proteina por kg de peso.<br/>En tu caso (~67kg): entre 107g y 147g al dia.<br/>Pollo, huevos, pescado y legumbres!"

    U->>LC: "Puedo hacer cardio en mis dias de descanso?"

    Note over A: Otra pregunta general<br/>→ NO tools necesarios

    A-->>U: "Claro! El cardio de baja intensidad<br/>(caminata, bici suave) en dias de descanso<br/>no afecta tu recuperacion muscular.<br/>20-30 min esta perfecto.<br/>Evita HIIT intenso los dias de descanso."
```

**Por que es mejor que n8n:** En n8n, CHAT va a un AI Agent generico sin contexto del usuario. Aqui el agente sabe que Camilo busca ganar masa y pesa 67kg — personaliza la respuesta.

---

## Caso 6: Renovacion de Mesociclo — Semana 4 Completada

> Ana completo todas las sesiones de semana 4. El agente detecta que puede renovar.
> Ana pide mantener pero cambiar de 4 a 5 dias.

```mermaid
sequenceDiagram
    participant U as Ana (WhatsApp)
    participant LC as load_context
    participant A as kairos_agent (Gemini)
    participant TN as ToolNode
    participant DB as Supabase

    Note over U,DB: Ana completo la ultima sesion de semana 4

    U->>LC: "Hola, que sigue ahora?"
    LC->>DB: SELECT users_plans, user_weekly_schedule
    DB-->>LC: plan: {mesocycle: 1, week: 4}, all_w4_completed: true
    LC-->>R: UserContext {<br/>  plan: {mesocycle_number: 1, week: 4, status: 'active'},<br/>  all_w4_completed: true<br/>}

    Note over A: System prompt incluye:<br/>"Semana 4/4 — COMPLETADA"<br/>"Mesociclo listo para renovacion"<br/>+ usuario pregunta "que sigue"<br/>→ Decide: llamar get_mesocycle_status

    A->>TN: tool_call: get_mesocycle_status(user_id)
    TN->>DB: SELECT users_plans, user_weekly_schedule WHERE week = 4
    DB-->>TN: { completed: true, mesocycle: 1,<br/>options: ['mantener', 'cambiar'] }
    TN-->>A: Mesociclo completado, puede renovar

    A-->>U: "Felicidades Ana! Completaste tu primer mesociclo!<br/>Tienes dos opciones:<br/>1. Mantener rutina (con progresion de carga)<br/>2. Cambiar ejercicios (nuevos estimulos)<br/>Que prefieres?"

    U->>LC: "Mantener pero quiero entrenar 5 dias ahora"

    Note over A: Quiere mantener + cambiar de 4 a 5 dias<br/>Esto requiere cambio week_schedule (ul_4 → ppl_5)<br/>→ Decide: informar el cambio de estructura

    A-->>U: "Genial! Para pasar a 5 dias usariamos<br/>un esquema Push-Pull-Legs.<br/>Mantenemos ejercicios similares pero<br/>con distribucion diferente.<br/>Confirmo el cambio a 5 dias/semana?"
```

**Por que es mejor que n8n:** En n8n, la renovacion requiere un subworkflow completo (GymBotMesocycleRenewal) con Wait nodes. El agente lo maneja en conversacion natural.

---

## Caso 7: Agendamiento Flexible

> Camilo tiene plan pero NO tiene dias programados.
> El agente detecta la falta de schedule y ayuda a programar.

```mermaid
sequenceDiagram
    participant U as Camilo (WhatsApp)
    participant LC as load_context
    participant A as kairos_agent (Gemini)
    participant TN as ToolNode
    participant DB as Supabase

    U->>LC: "Quiero programar mis entrenamientos"
    LC->>DB: SELECT users_plans, user_weekly_schedule
    DB-->>LC: plan activo (ul_4), schedule: []
    LC-->>R: UserContext {<br/>  plan: {week_schedule: 'ul_4', days_per_week: 4},<br/>  has_schedule: false<br/>}

    Note over A: Usuario quiere programar<br/>+ no tiene schedule<br/>→ Decide: llamar get_schedule_info

    A->>TN: tool_call: get_schedule_info(user_id)
    TN->>DB: SELECT users_plans JOIN week_schedules JOIN template_days
    DB-->>TN: { days_per_week: 4,<br/>sessions: ['Upper A', 'Lower A', 'Upper B', 'Lower B'] }
    TN-->>A: Info del plan

    A-->>U: "Tu plan tiene 4 sesiones por semana:<br/>• Upper Body A<br/>• Lower Body A<br/>• Upper Body B<br/>• Lower Body B<br/>Que dias prefieres entrenar?"

    U->>LC: "Lunes, miercoles, jueves y sabado"

    Note over A: Usuario dio 4 dias (coincide con 4 sesiones)<br/>→ Decide: llamar schedule_sessions

    A->>TN: tool_call: schedule_sessions(user_id, [<br/>  {week_day: 'Lunes', session: 'Upper A', day: '16/03'},<br/>  {week_day: 'Miercoles', session: 'Lower A', day: '18/03'},<br/>  {week_day: 'Jueves', session: 'Upper B', day: '19/03'},<br/>  {week_day: 'Sabado', session: 'Lower B', day: '21/03'}<br/>])
    TN->>DB: INSERT INTO user_weekly_schedule (4 rows)
    DB-->>TN: OK
    TN-->>A: "Schedule creado"

    A-->>U: "Listo! Tu horario queda asi:<br/>Lun → Upper Body A<br/>Mie → Lower Body A<br/>Jue → Upper Body B<br/>Sab → Lower Body B<br/>Arrancamos este lunes!"
```

---

## Caso 8: Creacion de Rutina — Draft Mode con Feedback

> Sofia acaba de completar el KYC. El agente le ofrece crear su rutina.
> Le pregunta como prefiere el proceso. Sofia elige ver todo junto.
> El agente genera un borrador, Sofia pide un cambio, el agente ajusta y guarda.

```mermaid
sequenceDiagram
    participant U as Sofia (WhatsApp)
    participant LC as load_context
    participant A as kairos_agent (Gemini)
    participant TN as ToolNode
    participant DB as Supabase

    Note over U,DB: Sofia termino KYC. Tiene perfil pero NO plan ni workouts.

    U->>LC: "Ya estoy lista, quiero mi rutina"
    LC->>DB: SELECT users, users_gym_profile, users_plans
    DB-->>LC: profile existe, plan: null
    LC-->>A: UserContext { kyc_complete: true, plan: null }

    Note over A: Tiene perfil pero no plan<br/>→ Decide: preguntar como prefiere la creacion

    A-->>U: "Voy a crear tu rutina de 4 dias!<br/>Como prefieres?<br/>1) Te la armo completa y te la muestro<br/>2) Vamos dia por dia para que opines"

    U->>LC: "Armala completa"

    Note over A: Prefiere ver todo junto<br/>→ Decide: llamar draft_routine

    A->>TN: tool_call: draft_routine(user_id, goal, level, week_schedule)
    TN->>DB: SELECT day_requirements JOIN template_days<br/>WHERE week_schedule = 'ul_4'
    TN->>DB: SELECT exercises WHERE pattern IN (...)<br/>AND level <= 'Intermedio'
    TN->>DB: SELECT set_profiles WHERE goal AND level
    DB-->>TN: { days: [<br/>  {name: 'Upper A', exercises: [Press banca, Remo, Militar, Curl]},<br/>  {name: 'Lower A', exercises: [Sentadilla, RDL, Prensa, Pantorrilla]},<br/>  {name: 'Upper B', exercises: [Press DB, Jalon, Laterales, Triceps]},<br/>  {name: 'Lower B', exercises: [Hip thrust, Leg curl, Ext quad, Core]}<br/>]}
    TN-->>A: Borrador completo (4 dias)

    A-->>U: "Tu rutina quedaria asi:<br/><br/>Upper A: Press banca, Remo barra, Militar, Curl<br/>Lower A: Sentadilla, RDL, Prensa, Pantorrilla<br/>Upper B: Press DB, Jalon, Laterales, Triceps<br/>Lower B: Hip thrust, Leg curl, Ext quad, Core<br/><br/>Que te parece? Cambio algo?"

    U->>LC: "No me gusta press militar, me duele el hombro"

    Note over A: Feedback: dolor de hombro + overhead<br/>→ Decide: buscar alternativa sin overhead

    A->>TN: tool_call: find_exercises(pattern="push_v",<br/>level="intermedio", exclude="overhead")
    TN->>DB: SELECT exercises WHERE pattern = 'push_v'<br/>AND level <= 'Intermedio'<br/>AND spanish_name NOT LIKE '%militar%'
    DB-->>TN: [Press inclinado DB, Landmine press, ...]
    TN-->>A: Alternativas disponibles

    Note over A: Selecciona la mejor opcion<br/>considerando hombro + perfil del usuario

    A-->>U: "Entendido! Te cambio press militar por<br/>press inclinado con mancuernas.<br/>Es mas amigable con los hombros.<br/><br/>Upper A actualizado:<br/>Press banca, Remo, Press inclinado DB, Curl<br/><br/>Asi esta bien?"

    U->>LC: "Si, perfecta!"

    Note over A: Aprobada<br/>→ Decide: guardar plan + workouts

    A->>TN: tool_call: save_workout_plan(user_id, draft_data)
    TN->>DB: INSERT users_plans (plan_id, template, goal, level)
    TN->>DB: INSERT workouts (bulk, 4 semanas x 4 dias)
    DB-->>TN: OK (plan + 64 workout rows created)
    TN-->>A: "Plan guardado"

    A-->>U: "Rutina guardada! 4 semanas de entrenamiento listas.<br/>Quieres programar tus dias ahora?"
```

### Variante: Dia por dia

```mermaid
sequenceDiagram
    participant U as Sofia (WhatsApp)
    participant A as kairos_agent (Gemini)
    participant TN as ToolNode

    U->>A: "Prefiero dia por dia"

    Note over A: Modo dia por dia<br/>→ draft_routine solo para dia 1

    A->>TN: draft_routine(user_id, day="Upper A")
    TN-->>A: Upper A: Press banca, Remo, Militar, Curl

    A-->>U: "Upper Body A:<br/>1. Press banca 4x8-10 RIR 2<br/>2. Remo barra 4x8-10 RIR 2<br/>3. Press militar 3x10-12 RIR 1<br/>4. Curl biceps 3x12-15<br/>Te parece bien?"

    U->>A: "Dale, siguiente"

    A->>TN: draft_routine(user_id, day="Lower A")
    TN-->>A: Lower A: Sentadilla, RDL, Prensa, Pantorrilla

    A-->>U: "Lower Body A:<br/>1. Sentadilla 4x8-10 RIR 2<br/>..."

    Note over U,A: [... repite para Upper B y Lower B ...]

    A-->>U: "Rutina completa! Guardo todo?"
    U->>A: "Si"
    A->>TN: save_workout_plan(user_id, draft_data)
```

### Variante: Cambio directo sin borrador

```mermaid
sequenceDiagram
    participant U as Sofia (WhatsApp)
    participant A as kairos_agent (Gemini)
    participant TN as ToolNode

    Note over U,A: Sofia ya tiene rutina del mesociclo 1

    U->>A: "Hacela igual que la anterior<br/>pero cambiame los curls por martillo"

    Note over A: Quiere la misma rutina con un swap<br/>→ No necesita draft completo

    A->>TN: swap_exercise(user_id, old="Curl biceps",<br/>new="Curl martillo")
    TN-->>A: Ejercicio reemplazado en borrador

    A-->>U: "Listo! Cambie curl biceps por curl martillo<br/>en toda la rutina. Guardo?"
```

**Por que es mejor que n8n:** En n8n, WORKOUT_CREATOR es una caja negra: entra perfil, sale rutina. El usuario no puede opinar durante la creacion. Aqui el agente presenta borradores, acepta feedback, y ajusta antes de guardar.

---

## Arquitectura General

```mermaid
flowchart TD
    subgraph DET["Deterministic Layer"]
        START((START)) --> LC[load_context<br/>Supabase queries]
        LC --> ROUTER{router}
    end

    subgraph AGENT_LAYER["Agentic Layer - Gemini"]
        ROUTER -->|new user| KYC[KYC Subgraph<br/>Case 5 - 7 nodos]
        ROUTER -->|existing user| KAIROS[kairos_agent<br/>Gemini + bind_tools]
        KAIROS -->|tool_calls?| TOOLNODE[ToolNode]
        TOOLNODE -->|result| KAIROS
        KAIROS -->|no tools| FORMAT[format_response]
    end

    subgraph TOOLS["Tools - Supabase"]
        TOOLNODE --> T1[get_todays_routine]
        TOOLNODE --> T2[confirm_workout]
        TOOLNODE --> T3[create_magic_link]
        TOOLNODE --> T4[schedule_sessions]
        TOOLNODE --> T5[get_mesocycle_status]
        TOOLNODE --> T6[draft_routine]
        TOOLNODE --> T7[save_workout_plan]
        TOOLNODE --> T8[find_exercises]
        TOOLNODE --> T9[swap_exercise]
        TOOLNODE --> T10[decline_workout]
        TOOLNODE --> T11[get_schedule_info]
    end

    KYC --> END_NODE((END))
    FORMAT --> END_NODE

    style DET fill:#dbe4ff,stroke:#4a9eed
    style AGENT_LAYER fill:#e5dbff,stroke:#8b5cf6
    style TOOLS fill:#d3f9d8,stroke:#22c55e
    style KYC fill:#ffd8a8,stroke:#f59e0b
    style KAIROS fill:#d0bfff,stroke:#8b5cf6
    style T6 fill:#fff3bf,stroke:#f59e0b
    style T7 fill:#fff3bf,stroke:#f59e0b
    style T8 fill:#fff3bf,stroke:#f59e0b
    style T9 fill:#fff3bf,stroke:#f59e0b
```

---

## Resumen: Como el Agente Decide

```mermaid
flowchart TD
    MSG[Mensaje del usuario] --> LC[load_context<br/>Supabase queries]
    LC --> CTX{Contexto cargado}

    CTX --> R{Router}
    R -->|nuevo + sin KYC| KYC[KYC Subgraph<br/>5 turnos, 10 campos]
    R -->|existente| AGENT[kairos_agent<br/>Gemini + Tools]

    AGENT --> DECIDE{Gemini analiza<br/>contexto + mensaje}

    DECIDE -->|"que me toca hoy?"<br/>+ tiene sesion| T1[get_todays_routine]
    DECIDE -->|"ya termine"<br/>+ sesion sin completar| T2[confirm_workout_completion]
    DECIDE -->|"pasame el link"| T3[create_magic_link]
    DECIDE -->|"quiero agendar"<br/>+ sin schedule| T4[get_schedule_info<br/>+ schedule_sessions]
    DECIDE -->|"que pasa con<br/>mi rutina?" + W4| T5[get_mesocycle_status]
    DECIDE -->|"cuanta proteina?"<br/>pregunta general| T6[Respuesta directa<br/>SIN tools]
    DECIDE -->|pending_task<br/>activa| T7[Pregunta primero<br/>luego tool segun resp]

    T1 --> RESP[Respuesta al usuario]
    T2 --> RESP
    T3 --> RESP
    T4 --> RESP
    T5 --> RESP
    T6 --> RESP
    T7 --> RESP

    style KYC fill:#f9d71c,color:#000
    style AGENT fill:#4ecdc4,color:#000
    style DECIDE fill:#ff6b6b,color:#fff
    style T6 fill:#95e1d3,color:#000
    style T7 fill:#f38181,color:#fff
```

## Punto Clave

El agente **NO tiene un Switch de intenciones**. Gemini recibe:
1. **Contexto** (plan, sesiones, tareas pendientes, semana del mesociclo)
2. **Mensaje** del usuario
3. **Tools** disponibles

Y **decide libremente** que hacer. Mismos mensajes pueden producir acciones diferentes segun el contexto:

| Mensaje | Contexto A | Contexto B |
|---------|-----------|-----------|
| "Hola" | Sin pending_task → saludo normal | Con pending_task → pregunta si completo rutina |
| "Ya termine" | Sesion hoy sin completar → `confirm_workout_completion` | Sin sesion hoy → "No tienes rutina programada hoy" |
| "Que sigue?" | Semana 2 → muestra rutina de manana | Semana 4 → `get_mesocycle_status` |
| "Quiero entrenar" | Sin schedule → ofrece agendar | Con schedule → muestra rutina de hoy |

---

## Agente vs n8n — Comparacion

| Escenario | n8n (Switch rigido) | Agente (Gemini + tools) |
|-----------|--------------------|-----------------------|
| Sesion perdida, quiere entrenar | "No tienes rutina hoy" | Ofrece sesion pendiente |
| Pending task + pregunta | Maneja pending task O pregunta, no ambos | Pregunta por pending, luego responde |
| Chat personalizado | AI Agent generico sin contexto | Responde con datos del perfil del usuario |
| Renovacion + cambio de dias | Requiere subworkflow completo | Conversacion natural, 2-3 mensajes |
| "Quiero el link del tracker" | No existe esta intencion | Agente entiende y genera magic link |
| Multiples intenciones en 1 msg | Solo detecta la primera | Razona sobre todo el mensaje |
