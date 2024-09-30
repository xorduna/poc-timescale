-- Crear l'extensió TimescaleDB si encara no està creada
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Crear la taula base
CREATE TABLE metrics (
       asset_id UUID NOT NULL,
       ts TIMESTAMPTZ NOT NULL,
       temp FLOAT,
       amb_humid FLOAT,
       setpoint FLOAT,
       coverage FLOAT,
       amb_temp FLOAT
);

-- Crear un índex compost en asset_id i ts
CREATE INDEX ON metrics (asset_id, ts DESC);

-- Convertir la taula en una hypertable
SELECT create_hypertable('metrics', 'ts');

-- Activar la compressió
ALTER TABLE metrics SET (
    timescaledb.compress,
    timescaledb.compress_orderby = 'ts DESC'
    );

-- Configurar la política de compressió
SELECT add_compression_policy('metrics', INTERVAL '7 days');