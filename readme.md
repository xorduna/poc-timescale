## Small Timescale POC ##

This project is a small POC to benchmark Timescale for an IoT scenario.

### How to run

Start timescale with PG16

```bash
make run-db
```

Timescale is listening at port 5432 with password `mysecretpassword`.

Execute model.sql inside postgres

Run the server (by default it will connect to local postgres)

```bash
./pts-server
```

You can use for example another database (for example a database in Timescale cloud)

```bash
./pts-server --dburi="postgres://tsdbadmin:<YOURPASSWORD>@<SERVICEID>.<PROJECTID>.tsdb.cloud.timescale.com:39377/tsdb?sslmode=require"
```

You can push some sample data with

```bash
curl -X POST http://localhost:8080/assets/<random-uuid>/metrics --data=@sample-payload
```

Now you can spawn as many clients you want

```bash
./pts-client
```

You can configure the interval, the batch sent and the number of fake devices to emulate

```bash
./pts-client --num=4 --interval=10s --batch=1m --api=http://server.aws.com
```

A PostgreSQL function is provided to fill the database with fake assets and fake data. Just run the SQL code in `faketelemetry.sql` to install the function and call the function to generate fake data

```sql
SELECT * FROM generate_fake_telemetry(
        100,                                -- 100 devices
        '2023-01-01 00:00:00'::TIMESTAMPTZ, -- Start date
        365,                                -- 365 days
        5                                   -- 5m interval
    );
```

And some handy functions to check database size

Chunk compression stats
```sql
SELECT 
    chunk_name, pg_size_pretty(before_compression_total_bytes), pg_size_pretty(after_compression_total_bytes) 
    FROM chunk_compression_stats('metrics')
    ORDER BY chunk_name;
  
```

Database compression stats
```sql
SELECT 
    pg_size_pretty(before_compression_total_bytes) as total_before, pg_size_pretty(after_compression_total_bytes) as total_after
    FROM hypertable_compression_stats('metrics');
```

Full database size stats
```sql
select pg_size_pretty(sum(pg_total_relation_size(oid))) FROM pg_class; 
select pg_size_pretty(sum(pg_database_size('tsdb')));
SELECT pg_size_pretty(SUM(size)) AS total_wal_size FROM pg_ls_waldir();
```