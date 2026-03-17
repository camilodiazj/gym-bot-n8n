# API Contract: Agente Unificado Kairos

**Feature**: `001-kairos-unified-agent`
**Date**: 2026-03-17
**Base URL**: `http://localhost:8000` (dev) / `https://[deploy-url]` (prod)

---

## POST /case6/chat

Enviar un mensaje al agente Kairos. El agente carga el contexto del usuario, decide qué herramientas usar, y responde.

### Request

```json
{
  "message": "que me toca hoy?",
  "phone_number": "573001234567",
  "display_name": "Camilo"
}
```

| Campo | Tipo | Requerido | Notas |
|-------|------|-----------|-------|
| `message` | `string` | Sí | Mensaje del usuario (WhatsApp) |
| `phone_number` | `string` | Sí | Número de teléfono completo con código de país |
| `display_name` | `string` | No | Nombre del contacto en WhatsApp (fallback si no existe en DB) |

### Response 200

```json
{
  "response": "Hoy te toca Upper Body A (Semana 2):\n1. Press banca 4x8-10 RIR 2 (150s)\n2. Remo con barra 4x8-10 RIR 2\n3. Press militar 3x10-12 RIR 1\n4. Curl bíceps 3x12-15\n¡Dale con todo!",
  "thread_id": "case6_573001234567",
  "is_new_user": false,
  "kyc_complete": true
}
```

| Campo | Tipo | Notas |
|-------|------|-------|
| `response` | `string` | Respuesta del agente para enviar al usuario por WhatsApp |
| `thread_id` | `string` | ID del thread de conversación (`case6_{phone_number}`) |
| `is_new_user` | `bool` | True si el usuario no existía en la base de datos |
| `kyc_complete` | `bool` | True si el perfil KYC está guardado |

### Response 422

```json
{
  "detail": "message and phone_number are required"
}
```

### Ejemplo — usuario nuevo (inicia KYC)

**Request**:
```json
{
  "message": "hola quiero empezar a entrenar",
  "phone_number": "579999999999"
}
```

**Response**:
```json
{
  "response": "¡Hola! Soy Kairos, tu entrenador personal. Para crear tu rutina necesito conocerte un poco. ¿Cuál es tu objetivo principal?",
  "thread_id": "case6_579999999999",
  "is_new_user": true,
  "kyc_complete": false
}
```

---

## GET /case6/history

Obtener el historial de conversación de un usuario.

### Request

```
GET /case6/history?phone_number=573001234567
```

| Param | Tipo | Requerido | Notas |
|-------|------|-----------|-------|
| `phone_number` | `string` | Sí | Número de teléfono |

### Response 200

```json
{
  "thread_id": "case6_573001234567",
  "message_count": 6,
  "messages": [
    {"role": "user", "content": "que me toca hoy?"},
    {"role": "assistant", "content": "Hoy te toca Upper Body A..."},
    {"role": "user", "content": "ya terminé mi rutina"},
    {"role": "assistant", "content": "Excelente Camilo! Upper Body A completada..."}
  ]
}
```

### Response 404

```json
{
  "detail": "No conversation found for phone_number 579000000001"
}
```

---

## Notas de integración

- El `thread_id` es `f"case6_{phone_number}"` — determinístico, no se genera aleatoriamente
- La conversación persiste en memoria mientras el servidor esté activo (InMemorySaver)
- Reiniciar el servidor limpia todos los threads — aceptable para el estado actual del proyecto
- El webhook de WhatsApp debe enviar `phone_number` en formato internacional sin `+` (ej: `573001234567`, no `+573001234567`)
- Mensajes de estado de WhatsApp (`message = ""` o `message_type = "status"`) deben filtrarse antes de llegar al endpoint
