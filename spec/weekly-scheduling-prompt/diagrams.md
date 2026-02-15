# Weekly Scheduling Prompt — Diagramas (KAN-61)

## 1. Diagrama de Flujo del Workflow

```mermaid
flowchart TD
    A["schedule_trigger_8pm<br/>Daily 20:00 COT"] --> B["query_users_needing_prompt<br/>Postgres CTE"]
    B --> C{"if_has_results<br/>items > 0?"}
    C -- FALSE --> END1["Fin<br/>(sin usuarios)"]
    C -- TRUE --> D["split_in_batches<br/>batchSize: 1"]
    D -- "Done<br/>(output 0)" --> END2["Fin<br/>(todos procesados)"]
    D -- "Each batch<br/>(output 1)" --> E{"if_full_completion<br/>completed == total?"}
    E -- TRUE --> G["set_celebration_msg<br/>Felicidades! Todas las sesiones"]
    E -- FALSE --> F{"if_zero_completion<br/>completed == 0?"}
    F -- TRUE --> I["set_reengagement_msg<br/>Quedo sin entrenamientos"]
    F -- FALSE --> H["set_growth_msg<br/>X de Y sesiones completadas"]
    G --> J["send_whatsapp<br/>WhatsApp Cloud API"]
    H --> J
    I --> J
    J -- "Loop back" --> D

    style A fill:#4A90D9,color:#fff
    style B fill:#7B68EE,color:#fff
    style G fill:#2ECC71,color:#fff
    style H fill:#F39C12,color:#fff
    style I fill:#E74C3C,color:#fff
    style J fill:#1ABC9C,color:#fff
```

## 2. Diagrama de Secuencia — Flujo Completo (Prompt + Respuesta)

```mermaid
sequenceDiagram
    participant Cron as Schedule Trigger<br/>(8 PM COT)
    participant PG as Supabase<br/>PostgreSQL
    participant WF as WeeklyScheduling<br/>Prompt Workflow
    participant WA as WhatsApp<br/>Cloud API
    participant User as Usuario
    participant MF as MAIN_FLOW
    participant Agent as AI Agent1<br/>(Scheduling)

    Note over Cron,WF: FASE 1: Deteccion y envio (automatico, 8 PM)

    Cron->>WF: Trigger diario 20:00
    WF->>PG: SELECT usuarios con semana terminada<br/>(weeks 1-3, ventana 3 dias, sin week+1)
    PG-->>WF: Lista de usuarios calificados

    alt Sin usuarios
        WF->>WF: Fin (sin accion)
    else Con usuarios
        loop Por cada usuario (batch=1)
            WF->>WF: Evaluar tasa de completitud
            alt completed == total
                WF->>WF: Mensaje Celebracion
            else completed == 0
                WF->>WF: Mensaje Re-engagement
            else 0 < completed < total
                WF->>WF: Mensaje Growth Mindset
            end
            WF->>WA: Enviar mensaje texto
            WA->>User: WhatsApp message
        end
    end

    Note over User,Agent: FASE 2: Respuesta del usuario (asincrono, horas/dias despues)

    User->>WA: "agendar"
    WA->>MF: Webhook WhatsApp
    MF->>MF: Intention Agent detecta: AGENDAR
    MF->>Agent: Redirigir a agente de programacion
    Agent->>User: "Para que dias quieres programar<br/>tu Semana X?"
    User->>Agent: "Lunes, miercoles y viernes"
    Agent->>PG: INSERT user_weekly_schedule<br/>(week+1 entries)
    Agent->>User: "Listo! Tu Semana X esta programada"

    Note over WF,PG: FASE 3: Dedup automatico (siguiente noche)

    Cron->>WF: Trigger diario 20:00
    WF->>PG: SELECT usuarios...
    PG-->>WF: Usuario YA NO aparece<br/>(NOT EXISTS week+1 lo excluye)
```

## 3. Diagrama de Flujo — Logica SQL de Deteccion

```mermaid
flowchart TD
    START["Todos los usuarios"] --> A{"Plan activo?<br/>users_plans.status = 'active'"}
    A -- No --> OUT1["Excluido"]
    A -- Si --> B{"Semana actual<br/>entre 1 y 3?"}
    B -- "No (week=4)" --> OUT2["Excluido<br/>(va a Mesocycle Renewal)"]
    B -- Si --> C{"Ultimo dia planeado<br/>< hoy COT?"}
    C -- No --> OUT3["Excluido<br/>(semana aun en curso)"]
    C -- Si --> D{"Ultimo dia planeado<br/>>= hoy - 3 dias?"}
    D -- No --> OUT4["Excluido<br/>(mas de 3 dias sin respuesta)"]
    D -- Si --> E{"Existe schedule<br/>para week+1?"}
    E -- Si --> OUT5["Excluido<br/>(ya programo siguiente semana)"]
    E -- No --> F{"DISTINCT ON user_id<br/>ORDER BY week DESC"}
    F --> QUALIFY["CALIFICADO<br/>Recibe prompt de scheduling"]

    style QUALIFY fill:#2ECC71,color:#fff
    style OUT1 fill:#95a5a6,color:#fff
    style OUT2 fill:#95a5a6,color:#fff
    style OUT3 fill:#95a5a6,color:#fff
    style OUT4 fill:#95a5a6,color:#fff
    style OUT5 fill:#95a5a6,color:#fff
```

## 4. Diagrama de Secuencia — Estrategia de Dedup

```mermaid
sequenceDiagram
    participant WF as Workflow
    participant DB as PostgreSQL

    Note over WF,DB: Dia 1: Usuario termina Semana 2 (3/3 sesiones)

    WF->>DB: Query usuarios calificados
    DB-->>WF: Juan (week=2, completed=3, total=3)
    WF->>WF: Envia mensaje Celebracion

    Note over WF,DB: Dia 2: Juan NO responde "agendar"

    WF->>DB: Query usuarios calificados
    Note right of DB: last_planned_day aun<br/>dentro de ventana 3 dias
    DB-->>WF: Juan (week=2, completed=3, total=3)
    WF->>WF: Envia mensaje Celebracion (2do intento)

    Note over WF,DB: Dia 3: Juan NO responde "agendar"

    WF->>DB: Query usuarios calificados
    Note right of DB: last_planned_day aun<br/>dentro de ventana 3 dias
    DB-->>WF: Juan (week=2, completed=3, total=3)
    WF->>WF: Envia mensaje Celebracion (3er intento)

    Note over WF,DB: Dia 4: Silencio automatico

    WF->>DB: Query usuarios calificados
    Note right of DB: last_planned_day >= today - 3<br/>YA NO SE CUMPLE
    DB-->>WF: (vacio - Juan excluido)
    WF->>WF: Fin. Juan no recibe mas mensajes.

    Note over WF,DB: Alternativa: Juan responde "agendar" en Dia 2

    WF->>DB: Query usuarios calificados
    Note right of DB: NOT EXISTS week=3<br/>YA NO SE CUMPLE<br/>(Juan tiene schedule week 3)
    DB-->>WF: (vacio - Juan excluido)
```

## 5. Mapa de Integracion entre Workflows

```mermaid
flowchart LR
    subgraph "8 PM Workflows"
        WSP["WeeklyScheduling<br/>Prompt<br/>(weeks 1-3)"]
        WC["GymBotWorkout<br/>Completion<br/>(confirmacion)"]
    end

    subgraph "5 AM Workflows"
        MR["MorningReminder<br/>(rutina del dia)"]
    end

    subgraph "Event-Driven"
        MF["MAIN_FLOW<br/>(WhatsApp webhook)"]
        MCR["GymBotMesocycle<br/>Renewal<br/>(week 4)"]
    end

    WSP -.->|"Usuario responde<br/>'agendar'"| MF
    MF -->|"AGENDAR intent"| MF
    MF -->|"Week 4 detected"| MCR

    subgraph "Tablas Compartidas"
        UWS[(user_weekly_schedule)]
        UP[(users_plans)]
        U[(users)]
    end

    WSP -->|"READ"| UWS
    WSP -->|"READ"| UP
    WSP -->|"READ"| U
    WC -->|"READ"| UWS
    MR -->|"READ"| UWS
    MF -->|"READ/WRITE"| UWS

    style WSP fill:#4A90D9,color:#fff
    style MF fill:#E67E22,color:#fff
    style MCR fill:#9B59B6,color:#fff
```

## 6. Tabla de Variantes de Mensaje

```mermaid
flowchart LR
    subgraph "Evaluacion"
        A{"completed_count<br/>vs<br/>total_sessions"}
    end

    subgraph "Celebration"
        B["completed == total<br/>---<br/>Felicidades! Completaste<br/>todas tus X sesiones"]
    end

    subgraph "Growth Mindset"
        C["0 < completed < total<br/>---<br/>Completaste X de Y<br/>sesiones. Cada<br/>entrenamiento suma"]
    end

    subgraph "Re-engagement"
        D["completed == 0<br/>---<br/>Veo que la semana<br/>quedo sin entrenamientos.<br/>Lo importante es volver"]
    end

    A -->|"100%"| B
    A -->|"1-99%"| C
    A -->|"0%"| D

    style B fill:#2ECC71,color:#fff
    style C fill:#F39C12,color:#fff
    style D fill:#E74C3C,color:#fff
```
