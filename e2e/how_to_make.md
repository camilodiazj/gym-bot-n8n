GUÍA DE ESTRATEGIA DE PRUEBAS E2E Y QA PARA FITBOT (n8n)
========================================================

1. EL PROBLEMA Y LA SOLUCIÓN
--------------------------------------------------------
El Problema: 
Al hacer cambios en el System Prompt o en la lógica de filtrado, corremos el riesgo de que el bot empiece a responder a mensajes de sistema (estados de "leído/enviado") o deje de responder a usuarios reales.

La Solución: 
Desacoplar el "Trigger" real (WhatsApp) de la lógica de negocio, permitiendo inyectar datos de prueba controlados (Mock Data) sin usar el celular.


2. ESTRATEGIA A: PRUEBAS RÁPIDAS (MANUAL MOCKING)
--------------------------------------------------------
Recomendado para validaciones rápidas antes de guardar cambios menores.

Cómo implementarlo:
1. Agrega un nodo "Manual Trigger" (o usa el botón "Test Workflow").
2. Agrega un nodo "Set" conectado al inicio de tu flujo (antes del filtro If).
3. Pega uno de los siguientes JSON en el nodo "Set" para simular la entrada.

--- JSON CASO 1: Simulación de "Ruido" (Status Update) ---
(El bot debe detenerse en el nodo If y NO generar respuesta)

{
  "messaging_product": "whatsapp",
  "metadata": {
    "display_phone_number": "573213413664",
    "phone_number_id": "914510145083991"
  },
  "statuses": [
    {
      "id": "wamid.HBgMNTczMjA4NzgwMDIwFQIAERgSMzc4NkZCRDNEOTBDRDA3NzY1AA==",
      "status": "sent",
      "timestamp": "1769084739",
      "recipient_id": "573208780020",
      "pricing": {
        "billable": false,
        "pricing_model": "PMP",
        "category": "service",
        "type": "free_customer_service"
      }
    }
  ],
  "field": "messages"
}

--- JSON CASO 2: Simulación de Usuario Real (Happy Path) ---
(El bot debe pasar el filtro y ejecutar el Agente IA)

{
  "messages": [
    {
      "from": "573001234567",
      "id": "wamid.TestID123",
      "timestamp": "1769085000",
      "text": {
        "body": "Hola, quiero una rutina para mejorar mi digestión y ganar fuerza."
      },
      "type": "text"
    }
  ]
}


3. ESTRATEGIA B: PRUEBAS AUTOMATIZADAS (EVALUATION NODES)
--------------------------------------------------------
Recomendado para asegurar estabilidad a largo plazo y validar cambios en el System Prompt. Esta estrategia utiliza los nodos nativos de n8n Evaluation (Beta).

Estructura del Workflow de Prueba:

1. Trigger (On new Evaluation event):
   - Inicia la prueba automáticamente.
   - Recibe una lista de casos de prueba (dataset).

2. Set Inputs:
   - Configura la entrada del flujo utilizando los datos del dataset.
   - Ejemplo: Asigna el campo 'JSON de entrada' a la variable del flujo principal.

3. [Tu Lógica de FitBot]:
   - Aquí se ejecuta tu cadena de nodos normal (If -> AI Agent).

4. Set Outputs:
   - Captura el resultado final.
   - Importante: Debes capturar si el flujo produjo una respuesta de texto o si terminó vacío.

5. Set Metrics (El Juez):
   - Define las reglas automáticas para aprobar o reprobar el cambio.

Ejemplos de Métricas:

A) Métrica: Bloqueo de Ruido
   - Condición: Si (Input tiene "statuses") Y (Output es undefined/vacío)
   - Resultado: PASS (1)

B) Métrica: Respuesta Correcta
   - Condición: Si (Input tiene "messages") Y (Output no está vacío)
   - Resultado: PASS (1)

C) Métrica: Alucinación (Fallo Crítico)
   - Condición: Si (Input tiene "statuses") Y (Output tiene texto)
   - Resultado: FAIL (0) -> ALERTA: El bot respondió a un mensaje técnico.


4. CHECKLIST DE DESPLIEGUE (PRE-FLIGHT CHECK)
--------------------------------------------------------
Antes de activar el workflow en producción (Switch a "Active"):

[ ] Prueba de Humo: 
    Ejecutar el caso de "Status Update" y verificar que el nodo If lo bloquee (False).

[ ] Prueba de Lógica: 
    Ejecutar una pregunta de fitness compleja para verificar que el nuevo System Prompt no alucina.

[ ] Limpieza: 
    Asegurarse de desconectar o borrar los nodos de prueba (Manual Trigger y Set) del flujo principal, o usar un nodo Switch que detecte si es entorno de ejecución manual.