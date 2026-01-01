#!/bin/bash
# Initialize replication user and slots on primary
# This runs after the main init scripts

set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Replication user for replicas
    CREATE USER replicator WITH REPLICATION ENCRYPTED PASSWORD 'replicator_password';
    
    -- Create replication slots (prevents WAL from being cleaned up before replicas consume it)
    SELECT pg_create_physical_replication_slot('replica1_slot');
    SELECT pg_create_physical_replication_slot('replica2_slot');
EOSQL

echo "Replication user and slots created"
