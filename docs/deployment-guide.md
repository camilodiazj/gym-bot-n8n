# Guía de Despliegue - GymBot Workout Tracker

## Arquitectura de Producción

| Componente | Servicio | URL |
|------------|----------|-----|
| Frontend | Firebase Hosting | https://workout-tracker-69b08.web.app |
| Backend API | Google Cloud Run | https://workout-api-148665080566.us-central1.run.app |
| Database | Supabase (PostgreSQL) | (ya configurado) |

---

## Requisitos Previos

```bash
# Instalar herramientas
brew install google-cloud-sdk
npm install -g firebase-tools

# Autenticarse
gcloud auth login
firebase login
```

---

## 1. Despliegue del Backend (Cloud Run)

### 1.1 Configurar variables de entorno

En la consola de Cloud Run o via CLI:

```bash
gcloud run services update workout-api \
  --region us-central1 \
  --set-env-vars "SUPABASE_DB_URL=postgresql://...,JWT_SECRET=tu-secret-32-chars,GIN_MODE=release"
```

### 1.2 Desplegar

```bash
cd workout-tracker-back

# Opción A: Deploy directo desde código fuente
gcloud run deploy workout-api \
  --source . \
  --region us-central1 \
  --allow-unauthenticated

# Opción B: Build local + deploy
docker build -t gcr.io/[PROJECT_ID]/workout-api .
docker push gcr.io/[PROJECT_ID]/workout-api
gcloud run deploy workout-api --image gcr.io/[PROJECT_ID]/workout-api --region us-central1
```

### 1.3 Verificar

```bash
curl https://workout-api-148665080566.us-central1.run.app/api/v1/health
# Debe retornar: {"success":true,"data":{"status":"ok"}}
```

---

## 2. Despliegue del Frontend (Firebase Hosting)

### 2.1 Configurar API URL

Editar `workout-tracker/src/App.tsx`:

```typescript
const API_BASE_URL = 'https://workout-api-148665080566.us-central1.run.app/api/v1'
```

### 2.2 Build y Deploy

```bash
cd workout-tracker

# Build de producción
npm run build

# Deploy a Firebase
firebase deploy --only hosting
```

### 2.3 Verificar

Abrir: https://workout-tracker-69b08.web.app

---

## 3. Configuración de n8n (WhatsApp Deep Links)

### 3.1 Variable de entorno

En n8n, configurar:
- `JWT_SECRET` = mismo valor que en Cloud Run

### 3.2 Importar workflow

Importar `n8n/running_flows/MorningReminder-WorkoutTracker.json`

### 3.3 Actualizar URL del frontend

En el nodo "Send WhatsApp", actualizar la URL:
```
https://workout-tracker-69b08.web.app/w?t={{ $json.token }}
```

---

## 4. Dominio Personalizado (Opcional)

### Firebase Hosting

```bash
firebase hosting:channel:deploy production
# Luego configurar dominio en Firebase Console > Hosting > Add custom domain
```

### Cloud Run

```bash
gcloud run domain-mappings create \
  --service workout-api \
  --domain api.gymbot.co \
  --region us-central1
```

---

## 5. Comandos Rápidos

### Re-deploy Backend
```bash
cd workout-tracker-back && gcloud run deploy workout-api --source . --region us-central1 --project gen-lang-client-0432163259
```

### Re-deploy Frontend
```bash
cd workout-tracker && npm run build && firebase deploy --only hosting
```

### Ver logs del Backend
```bash
gcloud run logs read workout-api --region us-central1 --limit 50
```

### Ver logs de Firebase
```bash
firebase hosting:channel:list
```

---

## 6. Troubleshooting

### Error CORS
Verificar que el middleware CORS en `router.go` permita el origen de Firebase:
```go
r.engine.Use(middleware.CORS())
```

### Error 401 en API
1. Verificar que `JWT_SECRET` sea igual en Cloud Run y n8n
2. Verificar que el token no haya expirado (24h)
3. Probar con `?user_id=` para desarrollo

### Frontend no carga datos
1. Verificar `API_BASE_URL` en `App.tsx`
2. Verificar en Network tab del browser que la petición llegue al backend
3. Revisar CORS headers en la respuesta

---

## 7. Costos Estimados

| Servicio | Free Tier | Uso típico |
|----------|-----------|------------|
| Cloud Run | 2M requests/mes | $0 |
| Firebase Hosting | 10GB storage, 360MB/día | $0 |
| Supabase | 500MB DB, 2GB bandwidth | $0 |

**Total estimado: $0/mes** (para ~100 usuarios activos)

---

## 8. Checklist de Deploy

- [ ] Backend desplegado en Cloud Run
- [ ] Variables de entorno configuradas (SUPABASE_DB_URL, JWT_SECRET)
- [ ] Health check responde OK
- [ ] Frontend desplegado en Firebase
- [ ] API_BASE_URL apunta a Cloud Run
- [ ] n8n workflow actualizado con nueva URL
- [ ] JWT_SECRET configurado en n8n
- [ ] Probado deep link desde WhatsApp
