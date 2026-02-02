/**
 * Parse_Intention Code Node
 *
 * Extrae la intencion del output del agente de renovacion.
 * Este codigo se usa en el nodo Code de n8n para parsear las respuestas
 * del Renewal_Agent y Profile_Modification_Agent.
 *
 * Location: GymBotMesocycleRenewal.json > Parse_Intention node
 */

// Extraer la intencion del output del agente de renovacion
const output = $input.first().json.output || '';
const inputData = $('Mesocycle_Renewal_Trigger').first().json;

// Patrones de intencion a buscar
const patterns = {
  mantener: /INTENCION:MANTENER_RUTINA/i,
  cambiarDias: /INTENCION:CAMBIAR_DIAS:(\d+)/i,
  rotar: /INTENCION:ROTAR_EJERCICIOS/i,
  modificarPerfil: /INTENCION:MODIFICAR_PERFIL/i,
  profileUpdate: /PROFILE_UPDATE:\s*(\{.*\})/i
};

let intention = 'PREGUNTAR_OPCIONES';
let newDays = null;
let profileUpdate = null;
let cleanOutput = output;

// Verificar cada patron
if (patterns.mantener.test(output)) {
  intention = 'MANTENER_RUTINA';
  cleanOutput = output.replace(patterns.mantener, '').trim();
}
else if (patterns.cambiarDias.test(output)) {
  const match = output.match(patterns.cambiarDias);
  intention = 'CAMBIAR_DIAS';
  newDays = parseInt(match[1]);

  // Validar rango
  if (newDays < 2) newDays = 2;
  if (newDays > 6) newDays = 6;

  cleanOutput = output.replace(patterns.cambiarDias, '').trim();
}
else if (patterns.rotar.test(output)) {
  intention = 'ROTAR_EJERCICIOS';
  cleanOutput = output.replace(patterns.rotar, '').trim();
}
else if (patterns.modificarPerfil.test(output)) {
  intention = 'MODIFICAR_PERFIL';
  cleanOutput = output.replace(patterns.modificarPerfil, '').trim();
}
else if (patterns.profileUpdate.test(output)) {
  intention = 'PROFILE_UPDATE_COMPLETE';
  const match = output.match(patterns.profileUpdate);
  try {
    profileUpdate = JSON.parse(match[1]);
  } catch (e) {
    // Si falla el parse, mantener como MODIFICAR_PERFIL para re-preguntar
    intention = 'MODIFICAR_PERFIL';
    profileUpdate = null;
  }
  cleanOutput = output.replace(patterns.profileUpdate, '').trim();
}

// Limpiar saltos de linea extras
cleanOutput = cleanOutput.replace(/\n{3,}/g, '\n\n').trim();

return [{
  json: {
    ...inputData,
    intention,
    newDays,
    profileUpdate,
    agentOutput: cleanOutput,
    rawOutput: output
  }
}];
