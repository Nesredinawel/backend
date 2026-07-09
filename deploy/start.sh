#!/bin/sh
# =========================
# Unified startup — runs all services in one container
# =========================
# Internal ports (can be overridden via env vars)
AUTH_PORT=${AUTH_PORT:-5001}
MOOD_PORT=${MOOD_PORT:-5002}
BLOG_PORT=${BLOG_PORT:-5003}
NOTIF_PORT=${NOTIF_PORT:-5004}
DOCS_PORT=${DOCS_PORT:-8085}      # hardcoded in api-docs-service
GW_PORT=${GW_PORT:-8081}          # hardcoded in api-gateway

# Construct Hasura GraphQL endpoint (from hostport → full URL)
if [ -n "$HASURA_GRAPHQL_ENDPOINT" ]; then
    export HASURA_GRAPHQL_ENDPOINT="http://${HASURA_GRAPHQL_ENDPOINT}/v1/graphql"
fi
export HASURA_GRAPHQL_ENDPOINT="${HASURA_GRAPHQL_ENDPOINT:-http://hasura:8080/v1/graphql}"

# Map env var name differences across services
export HASURA_ENDPOINT=${HASURA_GRAPHQL_ENDPOINT}

# Default Redis — override via Render dashboard env vars
export REDIS_ADDR=${REDIS_ADDR:-redis:6379}

# Each service uses a different env var for its port:
#   auth   → PORT_1    mood → PORT_2    blog → PORT_3
#   notif  → PORT      docs → hardcoded  gw   → hardcoded

PORT_1=$AUTH_PORT PORT_2=$MOOD_PORT PORT_3=$BLOG_PORT \
/usr/local/bin/auth-service &
echo "auth-service started on :$AUTH_PORT"

PORT_2=$MOOD_PORT PORT_1=$AUTH_PORT PORT_3=$BLOG_PORT \
/usr/local/bin/mood-service &
echo "mood-service started on :$MOOD_PORT"

PORT_3=$BLOG_PORT PORT_1=$AUTH_PORT PORT_2=$MOOD_PORT \
/usr/local/bin/blog-service &
echo "blog-service started on :$BLOG_PORT"

PORT=$NOTIF_PORT HASURA_ENDPOINT=$HASURA_GRAPHQL_ENDPOINT \
/usr/local/bin/notification-service &
echo "notification-service started on :$NOTIF_PORT"

/usr/local/bin/api-docs-service &
echo "api-docs-service started on :$DOCS_PORT"

# Point API gateway to localhost services
export AUTH_SERVICE_URL="http://localhost:$AUTH_PORT"
export MOOD_SERVICE_URL="http://localhost:$MOOD_PORT"
export BLOG_SERVICE_URL="http://localhost:$BLOG_PORT"
export NOTIFICATION_SERVICE_URL="http://localhost:$NOTIF_PORT"
export API_DOCS_SERVICE_URL="http://localhost:$DOCS_PORT"

echo "api-gateway starting on :$GW_PORT"
exec /usr/local/bin/api-gateway
