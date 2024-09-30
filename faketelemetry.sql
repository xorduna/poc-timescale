CREATE OR REPLACE FUNCTION generate_fake_telemetry(
    num_devices INT,
    start_date TIMESTAMP WITH TIME ZONE,
    num_days INT,
    interval_minutes INT,
    batch_size INT DEFAULT 1000
)
    RETURNS TABLE (asset_id UUID, row_count BIGINT) AS $$
DECLARE
    end_date TIMESTAMP WITH TIME ZONE;
    moment TIMESTAMP WITH TIME ZONE;
    device_ids UUID[];
    batch_start INT;
    batch_end INT;
    total_intervals INT;
BEGIN
    -- Genera la llista de UUIDs per als dispositius
    SELECT array_agg(gen_random_uuid())
    INTO device_ids
    FROM generate_series(1, num_devices);

    -- Calcula la data de finalització i el total d'intervals
    end_date := start_date + make_interval(days => num_days);
    total_intervals := CEIL(EXTRACT(EPOCH FROM (end_date - start_date)) / (interval_minutes * 60));

    -- Crea una taula temporal per emmagatzemar els recomptes
    CREATE TEMPORARY TABLE temp_counts (
                                           asset_id UUID,
                                           row_count BIGINT
    ) ON COMMIT DROP;

    -- Genera telemetries per al període especificat
            current_time := start_date;
    WHILE current_time < end_date LOOP
            -- Processa els dispositius en lots
            batch_start := 1;
            WHILE batch_start <= num_devices LOOP
                    batch_end := LEAST(batch_start + batch_size - 1, num_devices);

                    -- Inserció en batch per a l'interval actual i el lot de dispositius
                    INSERT INTO metrics (asset_id, ts, temp, amb_humid, setpoint, amb_temp, coverage)
                    SELECT
                        uuid,
                        current_time,
                        15 + random() * 15,
                        30 + random() * 40,
                        18 + random() * 7,
                        10 + random() * 25,
                        (30 + random() * 10) / 10
                    FROM unnest(device_ids[batch_start:batch_end]) AS uuid;

                    batch_start := batch_end + 1;
                END LOOP;

            -- Avança al següent interval
                    current_time := current_time + make_interval(mins => interval_minutes);
        END LOOP;

    -- Insereix els recomptes a la taula temporal
    INSERT INTO temp_counts (asset_id, row_count)
    SELECT unnest(device_ids), total_intervals::BIGINT;

    -- Retorna els resultats
    RETURN QUERY SELECT * FROM temp_counts;
END;
$$ LANGUAGE plpgsql;