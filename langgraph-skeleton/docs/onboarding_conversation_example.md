# Ejemplo de conversación — Onboarding reactivo

Flujo completo de un usuario nuevo, desde el saludo hasta rutina guardada y agendada.
Sirve como baseline de referencia para validar el comportamiento esperado de Kairos.

**Fecha del test**: 2026-03-23
**Revision**: `kairos-agent-00044-jb4`

---

## Turno 1 — Saludo

**Usuario:**
> Hola! Me recomendaron esta app para entrenar

**Kairos:**
> ¡Qué más, Andrés! Bienvenido a Kairos. Qué nota que te hayas animado a empezar a entrenar con nosotros, aquí le vamos a dar con toda para que alcances esos resultados que buscas.
>
> Para armarte un plan de entrenamiento bien bacano y a tu medida, cuéntame un par de cositas: **¿cuál es tu objetivo principal** (ganar músculo, bajar de peso, mejorar fuerza)? y **¿cuánta experiencia tienes entrenando?**
>
> Ah, y confírmame de una vez si vas a entrenar en el **gimnasio o en casa**. ¡Quedo atento!

**Qué hace bien:**
- Usa el nombre del `display_name` de WhatsApp sin preguntarlo
- Pregunta 3 datos en una sola pregunta natural (objetivo, experiencia, ambiente)
- NO pide email, peso, altura, edad ni datos innecesarios

---

## Turno 2 — Datos principales

**Usuario:**
> Quiero ganar músculo, llevo como un año yendo al gym, puedo ir 3 días a la semana

**Kairos:**
> ¡De una, Andrés! Con un año de experiencia y 3 días a la semana en el gym, podemos armar algo muy serio para que esa masa muscular empiece a notar el cambio.
>
> Solo me falta un detallito para dejarte listo: **¿tienes alguna lesión, dolor o condición de salud** que deba tener en cuenta para elegir tus ejercicios? (Por ejemplo, dolor de rodilla, espalda, etc.).
>
> ¡Apenas me digas, te registro y empezamos con tu rutina!

**Qué hace bien:**
- Registra 4 datos de un solo mensaje (goal, experience, days, environment)
- Solo pide el dato faltante: salud
- Explica para qué lo necesita ("para elegir tus ejercicios")

---

## Turno 3 — Salud + Registro + Draft de rutina

**Usuario:**
> A veces me molesta un poco la rodilla derecha pero nada grave

**Kairos:**
> ¡Listo, Andrés! Ya te registré en el sistema. Teniendo en cuenta que te molesta un poco la rodilla, he seleccionado ejercicios en máquina para los patrones de pierna, ya que ofrecen mayor estabilidad y suelen ser más amigables con las articulaciones mientras fortalecemos.
>
> Aquí tienes una propuesta de rutina **Full Body de 3 días** enfocada en ganar masa muscular:
>
> **Día 1 — Full Body A:**
> 1. Prensa de piernas (Quads) 3x8-10
> 2. Máquina de pecho / Chest press (Pecho) 3x8-10
> 3. Remo en máquina (Espalda) 3x8-10
> 4. Plancha (Abdomen) 3x12-15
>
> **Día 2 — Full Body B:**
> 1. Peso muerto / RDL (Isquios) 3x8-10
> 2. Jalón abierto (Espalda) 3x8-10
> 3. Press militar con mancuerna (Hombro) 3x8-10
> 4. Máquina de bíceps (Bíceps) 3x12-15
> 5. Abs General (Abdomen) 3x12-15
>
> **Día 3 — Full Body C:**
> 1. Sentadilla Hack (Quads) 3x8-10
> 2. Femoral sentado (Isquios) 3x12-15
> 3. Press inclinado con mancuerna (Pecho) 3x8-10
> 4. Jalón con triángulo (Espalda) 3x8-10
> 5. Curl en banca inclinada (Bíceps) 3x12-15
> 6. Abdominales colgado (Abdomen) 3x12-15
>
> ¿Qué te parece esta distribución? Si estás de acuerdo, la guardamos de una vez para que empieces.

**Qué hace bien:**
- Clasifica health como **código B** (rodilla)
- Llama `register_new_user` (crea user + perfil en una sola tool call)
- Procede directo a crear rutina sin turno extra
- Adapta ejercicios a la lesión (prensa en vez de sentadilla libre)
- Presenta draft completo con formato WhatsApp-friendly

**Tools llamadas:** `register_new_user` → `get_day_requirements` → `get_exercises_for_draft` (x varias)

---

## Turno 4 — Aprobación y guardado

**Usuario:**
> Se ve bien! Dale, guárdala

**Kairos:**
> ¡Listo, Andrés! Tu plan de entrenamiento para las próximas 4 semanas ya quedó guardado y configurado.
>
> Ahora, para que no se te pase ningún entrenamiento, **¿qué días de la semana vas a entrenar?** (Por ejemplo: Lunes, Miércoles y Viernes).
>
> Dime los días y de una vez te los agendo en el calendario. ¡Vamos con toda!

**Qué hace bien:**
- Llama `save_workout_plan` (60 ejercicios generados para 4 semanas)
- Sugiere agendar proactivamente

**Tools llamadas:** `save_workout_plan`

---

## Turno 5 — Agendar sesiones

**Usuario:**
> Lunes miércoles y viernes

**Kairos:**
> ¡Listo, Andrés! Ya quedaron agendadas tus sesiones de esta semana: **Lunes (hoy), Miércoles y Viernes**.
>
> Como hoy es lunes, ¡ya tienes tu primer entrenamiento esperándote! **¿Quieres que te muestre la rutina de hoy para que le des con toda?**

**Qué hace bien:**
- Llama `schedule_sessions` (3 sesiones en user_weekly_schedule)
- Ofrece mostrar rutina del día proactivamente

**Tools llamadas:** `get_schedule_info` → `schedule_sessions`

---

## Turno 6 — Ver rutina del día

**Usuario:**
> Sí, muéstrame la rutina de hoy

**Kairos:**
> ¡De una, Andrés! Aquí tienes tu rutina para hoy **Full Body A**. Dale con toda que hoy empezamos el camino a esos resultados:
>
> 1. **Prensa de piernas** (Quads): 3 sets x 8-10 reps (RIR 1-2, Descanso 150s)
> 2. **Máquina de pecho / Chest press** (Pecho): 3 sets x 8-10 reps (RIR 1-2, Descanso 150s)
> 3. **Remo en máquina** (Espalda): 3 sets x 8-10 reps (RIR 1-2, Descanso 150s)
> 4. **Plancha** (Abdomen): 3 sets x 12-15 reps (RIR 1-2, Descanso 60s)
>
> ¡Acuérdate de calentar bien antes de empezar! Apenas termines, me avisas por aquí para marcarla como completada. ¡A darle!

**Tools llamadas:** `get_todays_routine`

---

## Turno 7 — Confirmar entrenamiento

**Usuario:**
> Ya termine, estuvo brutal

**Kairos:**
> ¡Qué nota, Andrés! Me alegra mucho que te haya gustado, así es que se empieza con toda.
>
> Ya quedó registrado tu entrenamiento de hoy. Ahora a descansar y a comer bien para que esos músculos se recuperen. Nos vemos el **miércoles** para tu sesión de **Full Body B**. ¡A seguir así!

**Qué hace bien:**
- Llama `confirm_workout_completion`
- Da recomendación de recuperación
- Menciona próxima sesión

**Tools llamadas:** `confirm_workout_completion`

---

## Turno 8 — Email reactivo (sin email guardado)

**Usuario:**
> Oye me puedes enviar la rutina por correo?

**Kairos:**
> ¡Claro que sí, Andrés! Con gusto te la envío.
>
> Solo me falta un detallito: **¿cuál es tu correo electrónico?** No lo tengo registrado todavía. Apenas me lo pases, te mando tu rutina completa de una vez.

**Qué hace bien:**
- Llama `send_routine_email` → recibe error "no tiene email"
- Pide email al usuario naturalmente (no lo tenía porque no se preguntó en onboarding)

**Tools llamadas:** `send_routine_email` (falla) → pide dato al usuario

---

## Turno 9 — Guardar email y enviar

**Usuario:**
> andres.gomez.test@gmail.com

**Kairos:**
> ¡Listo, Andrés! Ya registré tu correo: andres.gomez.test@gmail.com.
>
> Te generé un **enlace mágico** para que entres a tu **Workout Tracker** personal. Ahí puedes ver tu rutina completa, registrar tus pesos y ver tus progresos:
>
> [Entrar a mi Workout Tracker](https://kairos-tracker.web.app/w?c=046491)

**Tools llamadas:** `update_user_profile` → `send_routine_email` (o fallback `create_magic_link`)

---

## Datos guardados en base de datos

| Campo | Valor | Fuente |
|-------|-------|--------|
| `full_name` | Andrés Gómez | WhatsApp display_name |
| `primary_goal` | Ganar masa muscular | Turno 2 |
| `training_experience` | 6 a 12 meses | Turno 2 ("como un año") |
| `days_available` | 3 | Turno 2 |
| `training_environment` | GYM | Turno 2 |
| `health_status` | B (rodilla) | Turno 3 (clasificado por LLM) |
| `fitness_level` | Intermedio | Derivado de experiencia |
| `email` | NULL → andres.gomez.test@gmail.com | Turno 9 (reactivo) |
| `age` | NULL | No pedido |
| `biological_sex` | NULL | No pedido |
| `height_cm` | NULL | No pedido |
| `weight_kg` | NULL | No pedido |

---

## Comparación con el flujo anterior

| Aspecto | KYC anterior (5 turnos) | Onboarding reactivo |
|---------|------------------------|---------------------|
| Turnos hasta rutina | 7+ (5 KYC + confirm + rutina) | **3** |
| Datos pedidos | 17 campos | **6 esenciales** |
| Datos innecesarios | 11 (peso, altura, sexo, cardio...) | **0** |
| Email | Pedido en turno 1 | **Solo cuando se necesita** |
| Tono | Formulario ("Pregunta 3 de 5") | **Conversacional natural** |
| Inferencia de contexto | Ninguna | **Infiere de texto/imágenes** |
| Adaptación a salud | Post-onboarding | **Inmediata en el draft** |
