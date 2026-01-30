#!/bin/bash
set -e

echo "🏗️  Building frontend..."
cd workout-tracker
npm ci
npm run build
cd ..

echo "📦 Copying frontend to backend static folder..."
rm -rf workout-tracker-back/static
cp -r workout-tracker/dist workout-tracker-back/static

echo "🚀 Deploying to Cloud Run..."
cd workout-tracker-back
gcloud run deploy workout-api \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars "GIN_MODE=release,CORS_ALLOWED_ORIGINS=https://workout-tracker-69b08.web.app"

echo "✅ Deploy complete!"
